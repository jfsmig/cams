//
// Created by jfs on 14/01/23.
//

#include "MediaEncoder.hpp"

MediaEncoder::MediaEncoder(StreamStorage &storage) : storage_{storage} {}

bool MediaEncoder::on_frame(const char *buf, size_t len) {
    return storage_.on_fragment(buf, len);
}
