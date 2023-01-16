//
// Created by jfs on 14/01/23.
//

#include "MediaDecoder.hpp"

#include <cassert>

#include "hub.pb.h"
#include "hub.grpc.pb.h"

// https://blog.kevmo314.com/custom-rtp-io-with-ffmpeg.html
// https://stackoverflow.com/questions/71000853/demuxing-and-decoding-raw-rtp-with-libavformat

static int read_packet(void *opaque, uint8_t *buf, int buf_size) {
    auto *stream = reinterpret_cast<::grpc::ServerReader<::cams::api::hub::DownstreamMediaFrame> *>(opaque);
    ::cams::api::hub::DownstreamMediaFrame frame;

    label_retry:
    frame.Clear();
    if (!stream->Read(&frame)) {
        // TODO(jfs) log
        return -1;
    }
    if (!frame.IsInitialized() || frame.IsInitializedWithErrors()) {
        // TODO(jfs) log
        return -1;
    }

    switch (frame.type()) {
        case ::cams::api::hub::DownstreamMediaFrameType::DOWNSTREAM_MEDIA_FRAME_TYPE_SDP:
            return -1;
        case ::cams::api::hub::DownstreamMediaFrameType::DOWNSTREAM_MEDIA_FRAME_TYPE_RTCP:
            // TODO(jfs) log
            goto label_retry;
        case ::cams::api::hub::DownstreamMediaFrameType::DOWNSTREAM_MEDIA_FRAME_TYPE_RTP:
            assert(buf_size > 0);
            if (frame.payload().size() > (size_t) buf_size) {
                // TODO(jfs) log
                return -1;
            }
            memcpy(buf, frame.payload().data(), frame.payload().size());
            return frame.payload().size();;
        default:
            // TODO(jfs) log
            return -1;
    }
}

static int write_packet(
        [[maybe_unused]] void *opaque,
        [[maybe_unused]] uint8_t *buf,
        [[maybe_unused]] int buf_size) { return -1; }

MediaDecoder::MediaDecoder(const std::string_view sdp, MediaEncoder &encoder) : encoder_{encoder} {
    int rc;

    input_format_context = avformat_alloc_context();
    assert(input_format_context != nullptr);

    input_format = av_find_input_format("sdp");
    assert(input_format != nullptr);

    rc = av_dict_set(&input_format_opts, "sdp_flags", "custom_io", 0);
    assert(rc == 0);

    rc = avformat_open_input(&input_format_context, "video.sdp", input_format, &input_format_opts);
    assert(rc == 0);

    avio_input_context_ = avio_alloc_context(
            readbuf_.data(), readbuf_.size(), 1,
            this, &read_packet, &write_packet, nullptr);
    assert(avio_input_context_ != nullptr);

    input_format_context->pb = avio_input_context_;
    input_format_context->flags |= AVFMT_FLAG_CUSTOM_IO;

    rc = avformat_open_input(&input_format_context, "/dev/null", input_format, nullptr);
    assert(rc == 0);
}

MediaDecoder::~MediaDecoder() {
    avformat_close_input(&input_format_context);
    avformat_free_context(input_format_context);
}

bool MediaDecoder::on_rtp(const char *buf, size_t len) {
    return encoder_.on_frame(buf, len);
}

bool MediaDecoder::on_rtcp(const char *buf, size_t len) {
    return encoder_.on_frame(buf, len);
}
