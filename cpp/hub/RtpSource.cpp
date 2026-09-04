// Copyright (c) 2022-2024 The authors (see the AUTHORS file)
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

//
// The RTP side of the pipeline.
//

#include "RtpSource.hpp"

#include <algorithm>
#include <cstring>
#include <utility>

extern "C" {
#include <libavutil/mem.h>
#include <libavutil/opt.h>
}

namespace {

// The description is a few hundred bytes; the buffer only has to make progress.
constexpr int kSdpBufferSize = 4096;

// RTP_MAX_PACKET_LENGTH in libavformat. The demuxer passes its own, larger
// buffer straight to the read callback, so this one only carries the receiver
// reports on the way out.
constexpr int kRtpBufferSize = 8192;

// What avformat_find_stream_info may consume looking for a picture header. The
// packets it reads are packets nothing else is waiting on yet, but they are
// still packets, so the bound is deliberately tight. Measured against the
// reference capture: the parser settles the dimensions after 21 source reads at
// this floor, and takes 64 at libavformat's own default of 5 MB.
constexpr int64_t kProbeBytes = 32 * 1024;
constexpr int64_t kAnalyzeMicros = AV_TIME_BASE;

void free_io(AVIOContext **io) {
    if (*io == nullptr) {
        return;
    }
    // avio_context_free does not free the buffer it was given.
    av_freep(&(*io)->buffer);
    avio_context_free(io);
}

} // namespace

RtpSource::RtpSource(std::string sdp, FrameSource &frames)
        : sdp_{std::move(sdp)}, frames_{frames} {}

RtpSource::~RtpSource() {
    if (ctx_ != nullptr) {
        // AVFMT_FLAG_CUSTOM_IO makes avformat_close_input leave pb alone, so
        // both contexts are ours to release.
        avformat_close_input(&ctx_);
    }
    free_io(&sdp_io_);
    free_io(&rtp_io_);
}

int RtpSource::read_sdp(void *opaque, uint8_t *buf, int len) {
    auto *self = static_cast<RtpSource *>(opaque);
    if (self->sdp_offset_ >= self->sdp_.size()) {
        return AVERROR_EOF;
    }
    const int n = static_cast<int>(
            std::min<std::size_t>(static_cast<std::size_t>(len),
                                  self->sdp_.size() - self->sdp_offset_));
    std::memcpy(buf, self->sdp_.data() + self->sdp_offset_, static_cast<std::size_t>(n));
    self->sdp_offset_ += static_cast<std::size_t>(n);
    return n;
}

int RtpSource::read_packet(void *opaque, uint8_t *buf, int len) {
    auto *self = static_cast<RtpSource *>(opaque);
    return self->frames_.next(buf, len);
}

int RtpSource::write_rtcp(void *opaque, const uint8_t *buf, int len) {
    auto *self = static_cast<RtpSource *>(opaque);
    (void) buf;

    // Discarded on purpose. libavformat emits a receiver report roughly twice a
    // second for whatever it is receiving, but the RTSP session belongs to the
    // agent and the hub has no path back to the camera. The alternative is a
    // read-only context, on which the same code would record a write error and
    // stop.
    self->rtcp_discarded_ += static_cast<uint64_t>(len);
    return len;
}

int RtpSource::open() {
    const AVInputFormat *fmt = av_find_input_format("sdp");
    if (fmt == nullptr) {
        return AVERROR_DEMUXER_NOT_FOUND;
    }

    ctx_ = avformat_alloc_context();
    if (ctx_ == nullptr) {
        return AVERROR(ENOMEM);
    }

    auto *sdp_buf = static_cast<uint8_t *>(av_malloc(kSdpBufferSize));
    if (sdp_buf == nullptr) {
        return AVERROR(ENOMEM);
    }
    sdp_io_ = avio_alloc_context(sdp_buf, kSdpBufferSize, 0, this,
                                 &RtpSource::read_sdp, nullptr, nullptr);
    if (sdp_io_ == nullptr) {
        av_free(sdp_buf);
        return AVERROR(ENOMEM);
    }
    // Presetting pb is also what sets AVFMT_FLAG_CUSTOM_IO, which is why the
    // contexts stay ours to free.
    ctx_->pb = sdp_io_;

    AVDictionary *opts = nullptr;
    // custom_io stops the demuxer opening sockets of its own. It would have
    // nowhere to point them anyway: a camera advertises "m=video 0" and
    // "c=IN IP4 0.0.0.0", naming neither an address nor a port.
    av_dict_set(&opts, "sdp_flags", "custom_io", 0);
    // The packets crossed a gRPC stream to get here, so they are already in
    // order. A reordering queue would only add latency and a path that reports
    // EAGAIN.
    av_dict_set(&opts, "reorder_queue_size", "0", 0);
    ctx_->max_delay = 0;

    // Naming the format skips probing, which would otherwise consume the
    // description looking for something to recognise.
    int rc = avformat_open_input(&ctx_, "cams.sdp", fmt, &opts);
    av_dict_free(&opts);
    if (rc < 0) {
        // avformat_open_input frees and nulls the context on failure.
        ctx_ = nullptr;
        return rc;
    }

    auto *rtp_buf = static_cast<uint8_t *>(av_malloc(kRtpBufferSize));
    if (rtp_buf == nullptr) {
        return AVERROR(ENOMEM);
    }
    rtp_io_ = avio_alloc_context(rtp_buf, kRtpBufferSize, 1, this,
                                 &RtpSource::read_packet, &RtpSource::write_rtcp,
                                 nullptr);
    if (rtp_io_ == nullptr) {
        av_free(rtp_buf);
        return AVERROR(ENOMEM);
    }

    // The description has been consumed. From here the same field carries the
    // packets, and its context is the write-capable one.
    ctx_->pb = rtp_io_;
    free_io(&sdp_io_);

    // The description gives the depacketiser the codec and the SPS/PPS out of
    // "sprop-parameter-sets", but not the picture dimensions: nothing there has
    // parsed an SPS. A fragmented-MP4 track cannot be declared without them --
    // libavformat refuses one with "dimensions not set" -- so they are settled
    // here, once, instead of at the sink.
    //
    // This is what makes open() read packets, where it used to promise it did
    // not. avformat_find_stream_info runs libavcodec's H.264 parser, which also
    // supplies the profile and level; it tolerates the EAGAIN the demuxer
    // returns for an incomplete access unit, and the packets it buffers are
    // handed back by av_read_frame afterwards, so nothing is lost.
    ctx_->probesize = kProbeBytes;
    ctx_->max_analyze_duration = kAnalyzeMicros;
    rc = avformat_find_stream_info(ctx_, nullptr);
    if (rc < 0) {
        return rc;
    }
    return 0;
}

int RtpSource::read(AVPacket *pkt) {
    for (;;) {
        const int rc = av_read_frame(ctx_, pkt);
        if (rc != AVERROR(EAGAIN)) {
            return rc;
        }
        // EAGAIN here means a packet was consumed without completing an access
        // unit -- an RTCP report, or a fragment in the middle of one. The read
        // callback blocks, so each round consumed something and this makes
        // progress.
    }
}
