//
// Created by jfs on 14/01/23.
//

#pragma once

#include <string>

#include "Uncopyable.hpp"
#include "StreamStorage.hpp"
#include "RTP.hpp"

class MediaEncoder : Uncopyable {
public:
    ~MediaEncoder() = default;

    MediaEncoder() = delete;

    explicit MediaEncoder(StreamStorage &storage);

    bool on_frame(const RtpPacket &);

    void flush();

private:
    StreamStorage &storage_;
};
