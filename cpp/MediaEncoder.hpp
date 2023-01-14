//
// Created by jfs on 14/01/23.
//

#ifndef CAMS_CPP_MEDIAENCODER_HPP
#define CAMS_CPP_MEDIAENCODER_HPP

#include <string>

#include "Uncopyable.hpp"
#include "StreamStorage.hpp"

class MediaEncoder : Uncopyable {
public:
    ~MediaEncoder() = default;

    MediaEncoder() = delete;

    MediaEncoder(StreamStorage &storage) : storage_{storage} {

    }

    void on_frame(const char *buf, size_t len) {

    }

private:
    StreamStorage &storage_;
};


#endif //CAMS_CPP_MEDIAENCODER_HPP
