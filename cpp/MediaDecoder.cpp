//
// Created by jfs on 14/01/23.
//

#include "MediaDecoder.hpp"

#include <cassert>
#include <unistd.h>

// https://blog.kevmo314.com/custom-rtp-io-with-ffmpeg.html
// https://stackoverflow.com/questions/71000853/demuxing-and-decoding-raw-rtp-with-libavformat

static int read_packet(void *opaque, uint8_t *buf, int buf_size) {
    return reinterpret_cast<MediaSource*>(opaque)->Read(buf, buf_size);
}

static int write_packet(
        [[maybe_unused]] void *opaque,
        [[maybe_unused]] uint8_t *buf,
        [[maybe_unused]] int buf_size) { return -1; }

MediaDecoder::MediaDecoder(const std::string_view sdp,
                           MediaSource &source,
                           MediaEncoder &encoder) : encoder_{encoder} {
    int rc;

    input_format_context = avformat_alloc_context();
    assert(input_format_context != nullptr);

    input_format = av_find_input_format("sdp");
    assert(input_format != nullptr);

    rc = av_dict_set(&input_format_opts, "sdp_flags", "custom_io", 0);
    assert(rc == 0);

    char tmppath[] = "/tmp/sdp-XXXXXX";
    int fd = mkstemp(tmppath);
    rc = write(fd, sdp.data(), sdp.size());
    assert(rc == sdp.size());

    rc = avformat_open_input(&input_format_context, tmppath, input_format, &input_format_opts);
    assert(rc == 0);

    avio_input_context_ = avio_alloc_context(
            readbuf_.data(), readbuf_.size(), 1,
            &source_, &read_packet, &write_packet, nullptr);
    avio_input_context_->direct = 1;
    assert(avio_input_context_ != nullptr);

    input_format_context->pb = avio_input_context_;
    input_format_context->flags |= AVFMT_FLAG_CUSTOM_IO;

    rc = avformat_open_input(&input_format_context, "/dev/null", input_format, nullptr);
    assert(rc == 0);

    (void) unlink(tmppath);
    (void) close(fd);
    fd = -1;
}

MediaDecoder::~MediaDecoder() {
    avformat_close_input(&input_format_context);
    avformat_free_context(input_format_context);
}

bool MediaDecoder::on_rtp(const char *buf, size_t len) {
    return encoder_.on_frame(buf, len);
}
