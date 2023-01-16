//
// Created by jfs on 16/01/23.
//

#ifndef CAMS_CPP_MEDIASOURCE_HPP
#define CAMS_CPP_MEDIASOURCE_HPP

#include <cstdint>
#include <cstddef>

class MediaSource {
public:
    MediaSource() = default;

    virtual ~MediaSource() = default;

    virtual int Read(uint8_t *buf, size_t buf_len) = 0;
};


#endif //CAMS_CPP_MEDIASOURCE_HPP
