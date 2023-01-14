//
// Created by jfs on 14/01/23.
//

#ifndef CAMS_CPP_MEDIADECODER_HPP
#define CAMS_CPP_MEDIADECODER_HPP

#include <string>

#include "Uncopyable.hpp"
#include "MediaEncoder.hpp"

class MediaDecoder : Uncopyable {
public:
    ~MediaDecoder() = default;

    MediaDecoder() = delete;

    MediaDecoder(const std::string_view sdp, MediaEncoder &encoder) : encoder_{encoder} {

    }

    void onRTP(const char *buf, size_t len) {

    }

    void onRTCP(const char *buf, size_t len) {

    }

private:
    MediaEncoder &encoder_;
};


#endif //CAMS_CPP_MEDIADECODER_HPP
