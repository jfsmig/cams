//
// Created by jfs on 14/01/23.
//

#include "StreamStorage.hpp"

StreamStorage::StreamStorage(const std::string &user, const std::string &camera) {
    (void) user, (void) camera;
}

bool StreamStorage::on_fragment(const char *buf, size_t len) {
    (void) buf, (void) len;
    return false;
}

void StreamStorage::flush () {}
