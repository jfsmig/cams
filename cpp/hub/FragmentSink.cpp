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
// The HLS side of the pipeline: fragments out, and the playlist naming them.
//

#include "FragmentSink.hpp"

#include <algorithm>
#include <cstdio>
#include <string>
#include <utility>

extern "C" {
#include <libavutil/opt.h>
}

namespace {

// How many fragments past the end of the playlist stay on disk. See open().
constexpr int64_t kDeleteThreshold = 1;

} // namespace

FragmentSink::FragmentSink(const StreamStorage &storage, HlsOptions opts,
                           StreamStorage::Continuity how)
        : storage_{storage}, opts_{opts}, how_{how} {}

FragmentSink::~FragmentSink() {
    if (ctx_ != nullptr) {
        // The HLS muxer is AVFMT_NOFILE: it opened every file itself and there
        // is no AVIOContext of ours to close.
        avformat_free_context(ctx_);
        ctx_ = nullptr;
    }
}

int FragmentSink::open(const AVCodecParameters *in, AVRational in_tb) {
    in_tb_ = in_tb;

    int rc = avformat_alloc_output_context2(&ctx_, nullptr, "hls",
                                            storage_.playlist().c_str());
    if (rc < 0) {
        return rc;
    }

    AVStream *out = avformat_new_stream(ctx_, nullptr);
    if (out == nullptr) {
        return AVERROR(ENOMEM);
    }
    rc = avcodec_parameters_copy(out->codecpar, in);
    if (rc < 0) {
        return rc;
    }
    // The tag identified the codec inside the container the parameters came
    // from, which was RTP. It means nothing in MP4, so the muxer picks one --
    // except for HEVC, where the two it can pick between are not equivalent to
    // a player.
    //
    // hvc1 rather than the hev1 the muxer defaults to: hvc1 is what Safari and
    // QuickTime will play, and Safari is most of the reason to carry HEVC at
    // all. Strictly, hvc1 says the parameter sets live only in the sample
    // description while hev1 allows them in band as well, and a camera does
    // send them in band -- but every player that accepts HEVC in fMP4 accepts
    // the in-band copies, and the ones that matter here reject hev1. This
    // is the one choice in the H.265 arm not verified against a real player.
    out->codecpar->codec_tag =
            in->codec_id == AV_CODEC_ID_HEVC ? MKTAG('h', 'v', 'c', '1') : 0;

    // Checked here rather than left to the muxer, which reports it as a bare
    // "dimensions not set" with nothing naming the stream.
    if (out->codecpar->width <= 0 || out->codecpar->height <= 0) {
        fprintf(stderr,
                "cams: the stream reports %dx%d, so no fMP4 track can be "
                "declared\n",
                out->codecpar->width, out->codecpar->height);
        return AVERROR(EINVAL);
    }
    // A hint only: writing the header replaces it with the inner container's.
    out->time_base = in_tb_;

    void *pd = ctx_->priv_data;
    // hls_time is an AV_OPT_TYPE_DURATION, which is microseconds.
    rc = av_opt_set_int(pd, "hls_time",
                        static_cast<int64_t>(opts_.segment_seconds) * 1000000, 0);
    if (rc < 0) {
        return rc;
    }
    // hls_list_size is a count of fragments, so the retention is stated once in
    // seconds and converted here. At least one, because a playlist listing
    // nothing is not a playlist.
    const int64_t list_size =
            opts_.keep_all
                    ? 0
                    : std::max(1, opts_.retention_seconds / opts_.segment_seconds);
    rc = av_opt_set_int(pd, "hls_list_size", list_size, 0);
    if (rc < 0) {
        return rc;
    }
    // Set to libavformat's own default rather than left implicit, because the
    // retention arithmetic depends on it: a fragment that falls off the
    // playlist survives one more round before being unlinked, so the directory
    // holds one fragment beyond what the playlist names. That round is not
    // waste -- a player that fetched the playlist a moment earlier is still
    // entitled to the fragment that has just dropped off it.
    rc = av_opt_set_int(pd, "hls_delete_threshold", kDeleteThreshold, 0);
    if (rc < 0) {
        return rc;
    }
    rc = av_opt_set(pd, "hls_segment_type", "fmp4", 0);
    if (rc < 0) {
        return rc;
    }
    rc = av_opt_set(pd, "hls_fmp4_init_filename", storage_.init_name().c_str(), 0);
    if (rc < 0) {
        return rc;
    }
    rc = av_opt_set(pd, "hls_segment_filename", storage_.segments().c_str(), 0);
    if (rc < 0) {
        return rc;
    }
    // independent_segments because every fragment starts on a keyframe here;
    // temp_file so a player never fetches one that is still being written; and
    // program_date_time because it is what lets a viewer seek to the wall-clock
    // moment an event was reported, which is the whole point of keeping the
    // fragments.
    std::string flags = "independent_segments+temp_file+program_date_time";
    if (how_ == StreamStorage::Continuity::Append) {
        // Without this the muxer truncates the playlist and restarts fragment
        // numbering at zero, so every reconnect -- a camera reboot, a network
        // blip, the agent's own retry -- overwrote the recording it was meant
        // to extend. With it, the sequence continues and the muxer marks the
        // join with an EXT-X-DISCONTINUITY, which is honest: the new session is
        // a new encoder with its own clock.
        flags += "+append_list";
    }
    if (!opts_.keep_all) {
        // What falls off the playlist is deleted, which is what makes
        // retention_seconds a bound on the directory and not just on the
        // playlist.
        flags += "+delete_segments";
    }
    rc = av_opt_set(pd, "hls_flags", flags.c_str(), 0);
    if (rc < 0) {
        return rc;
    }

    rc = avformat_write_header(ctx_, nullptr);
    if (rc < 0) {
        return rc;
    }
    opened_ = true;
    return 0;
}

int FragmentSink::write(AVPacket *pkt) {
    pkt->stream_index = 0;
    // Rescaled against what the muxer settled on, not what we asked for:
    // writing the header replaced the stream's time base with the inner
    // container's. Reading back the value we set instead would misjudge every
    // segment boundary by the ratio between the two.
    av_packet_rescale_ts(pkt, in_tb_, ctx_->streams[0]->time_base);
    pkt->pos = -1;

    const int rc = av_interleaved_write_frame(ctx_, pkt);
    if (rc >= 0) {
        written_++;
    }
    return rc;
}

int FragmentSink::finish() {
    int rc = 0;
    if (opened_) {
        rc = av_write_trailer(ctx_);
        opened_ = false;
    }
    return rc;
}
