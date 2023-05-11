//
// Created by jfs on 14/01/23.
//

#pragma once

#include <array>
#include <string>

extern "C" {
#include <libavformat/avformat.h>
#include <libavcodec/avcodec.h>
#include <libavutil/opt.h>
}

#include "Uncopyable.hpp"
#include "MediaEncoder.hpp"

class MediaDecoder : Uncopyable {
public:
    MediaDecoder() = delete;

    MediaDecoder(const std::string_view sdp, MediaEncoder &encoder);

    ~MediaDecoder();

    bool on_rtp(const char *buf, size_t len);

private:
    // Input
    MediaSource &source_;

    // Output
    MediaEncoder &encoder_;

    const AVInputFormat *input_format = nullptr;
    AVFormatContext *input_format_context = nullptr;
    AVDictionary *input_format_opts = nullptr;
    AVIOContext * avio_input_context_ = nullptr;
    std::array<uint8_t, 8192> readbuf_;
};
