//
// Created by jfs on 16/01/23.
//

#include <iostream>

#include "StreamStorage.hpp"
#include "MediaSource.hpp"
#include "MediaEncoder.hpp"
#include "MediaDecoder.hpp"

class DevNullSource : public MediaSource {
public:
    DevNullSource() = default;
    ~DevNullSource() override = default;

    int Read(uint8_t *buf, size_t buf_size) override {
        (void) buf, (void) buf_size;
        std::cout << __func__ << " " << __FILE__ << " +" << __LINE__<< std::endl;
        return AVERROR_EXIT;
    }
};

int main(int argc, char **argv) {
    (void) argc, (void) argv;

    const char *sdp = R"(v=0
o=- 0 0 IN IP4 127.0.0.1
s=No Name
c=IN IP4 127.0.0.1
t=0 0
a=tool:libavformat 58.29.100
m=video 5000 RTP/AVP 96
a=rtpmap:96 H264/90000
a=fmtp:96 level-asymmetry-allowed=1;packetization-mode=1;profile-level-id=42001f)";

    StreamStorage storage("user", "camera");
    MediaEncoder encoder(storage);
    DevNullSource source;
    [[maybe_unused]] MediaDecoder decoder(sdp, source, encoder);

    return 0;
}