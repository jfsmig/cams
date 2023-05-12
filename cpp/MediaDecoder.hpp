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

    MediaDecoder(std::string_view sdp, MediaEncoder &encoder);

    ~MediaDecoder();

    bool on_rtp(const uint8_t *buf, size_t len);

private:
    // Output
    MediaEncoder &encoder_;

    AVFormatContext *input_format_context_ = nullptr;
    AVDictionary *input_format_opts_ = nullptr;
    std::array<uint8_t, 32768> readbuf_;
};
