//
// Created by jfs on 14/01/23.
//

#ifndef CAMS_CPP_STREAMSTORAGE_HPP
#define CAMS_CPP_STREAMSTORAGE_HPP

#include <string>

#include "Uncopyable.hpp"

class StreamStorage : Uncopyable {
public:
    ~StreamStorage() = default;

    StreamStorage() = delete;

    StreamStorage(const std::string &user, const std::string &camera);

    bool on_fragment(const char *buf, size_t len);

private:
};


#endif //CAMS_CPP_STREAMSTORAGE_HPP
