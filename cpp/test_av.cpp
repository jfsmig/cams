//
// Created by jfs on 16/01/23.
//

#include "StreamStorage.hpp"
#include "MediaEncoder.hpp"
#include "MediaDecoder.hpp"

int main(int argc, char **argv) {
    (void) argc, (void) argv;

    const char *sdp = "";

    StreamStorage storage("user", "camera");
    MediaEncoder encoder(storage);
    [[maybe_unused]] MediaDecoder decoder(sdp, encoder);

    return 0;
}