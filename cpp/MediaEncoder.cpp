//
// Created by jfs on 14/01/23.
//

#include "MediaEncoder.hpp"

MediaEncoder::MediaEncoder(StreamStorage &storage) : storage_{storage} {}

bool MediaEncoder::on_frame(const char *uint8_t, size_t len) {
    return storage_.on_fragment(buf, len);
}

void MediaEncoder::flush() {
    storage_.flush();
}