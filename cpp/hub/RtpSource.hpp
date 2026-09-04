// Copyright (c) 2022-2024 The authors (see the AUTHORS file)
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

//
// The RTP side of the pipeline: a session description and a stream of packets
// in, demuxed H.264 access units out.
//

#pragma once

#include <cstdint>
#include <string>

extern "C" {
#include <libavformat/avformat.h>
#include <libavformat/avio.h>
}

#include "FrameSource.hpp"
#include "Uncopyable.hpp"

// RtpSource depacketises RTP with libavformat's own "sdp" demuxer.
//
// It replaces about 260 lines that parsed RTP headers and scanned for Annex-B
// start codes by hand. That could not have worked: an H.264 RTP payload carries
// no start codes at all, only a single NAL unit, a STAP-A aggregate or an FU-A
// fragment (RFC 6184), and the reference capture is almost entirely FU-A. What
// libavformat's rtpdec_h264 does instead is the whole of RFC 6184
// non-interleaved mode, plus the SPS and PPS out of "sprop-parameter-sets", plus
// unwrapping the 32-bit RTP timestamps into a monotonic presentation clock.
//
// The awkwardness of the arrangement is entirely in the plumbing, and it is
// worth naming because it is not obvious from the API:
//
//   * the description is handed to the demuxer through AVFormatContext::pb, the
//     same field the packets then arrive on. They cannot share one AVIOContext:
//     reading the description to EOF latches eof_reached, and the recovery path
//     consumes the first packet while reporting a spurious end of file;
//   * so there are two, and the second is opened write-capable. That makes
//     avio_read_partial hand the demuxer whatever a single read returns, rather
//     than buffering across packet boundaries, and it gives libavformat
//     somewhere to put the RTCP receiver reports it insists on generating.
class RtpSource : Uncopyable {
public:
    RtpSource() = delete;

    // sdp must already be reduced to the media that frames will carry; see
    // keep_first_video_media.
    RtpSource(std::string sdp, FrameSource &frames);

    ~RtpSource();

    // Parses the description, prepares the demuxer, and settles the stream
    // parameters -- which includes the picture dimensions the fMP4 muxer
    // requires and the description does not carry.
    //
    // It therefore reads packets and blocks: about twenty of them against the
    // reference capture, a fraction of a second of stream. Returns 0, or a
    // libav error code.
    [[nodiscard]] int open();

    [[nodiscard]] AVFormatContext *context() const { return ctx_; }

    // Reads the next access unit. AVERROR_EOF once the source is exhausted.
    [[nodiscard]] int read(AVPacket *pkt);

    // How many bytes of receiver report libavformat generated and this
    // discarded.
    [[nodiscard]] uint64_t rtcp_discarded() const { return rtcp_discarded_; }

private:
    static int read_sdp(void *opaque, uint8_t *buf, int len);
    static int read_packet(void *opaque, uint8_t *buf, int len);
    static int write_rtcp(void *opaque, const uint8_t *buf, int len);

    std::string sdp_;
    std::size_t sdp_offset_{0};
    FrameSource &frames_;

    AVFormatContext *ctx_{nullptr};
    AVIOContext *sdp_io_{nullptr};
    AVIOContext *rtp_io_{nullptr};
    uint64_t rtcp_discarded_{0};
};
