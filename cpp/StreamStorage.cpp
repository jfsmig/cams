//
// Created by jfs on 14/01/23.
//

#include "StreamStorage.hpp"

StreamStorage::StreamStorage(const std::string &user, const std::string &camera) {}

bool StreamStorage::on_fragment(const char *buf, size_t len) {
    return false;
}
