//
// Created by jfs on 14/01/23.
//

#include "NAL.hpp"

#include "MediaEncoder.hpp"

MediaEncoder::MediaEncoder(StreamStorage &storage) : storage_{storage} {}

bool MediaEncoder::on_frame(const MediaFrame &frame) {
    return true;
}

void MediaEncoder::flush() {
    storage_.flush();
}