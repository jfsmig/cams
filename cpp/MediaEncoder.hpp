//
// Created by jfs on 14/01/23.
//

#pragma once

#include <string>

#include "Uncopyable.hpp"
#include "StreamStorage.hpp"

class MediaEncoder : Uncopyable {
public:
    ~MediaEncoder() = default;

    MediaEncoder() = delete;

    explicit MediaEncoder(StreamStorage &storage);

    bool on_frame(const uint8_t *buf, size_t len);

    void flush();

private:
    StreamStorage &storage_;
};
