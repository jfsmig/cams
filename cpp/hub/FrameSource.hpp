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
// Where the RTP demuxer pulls its packets from.
//

#pragma once

#include <cstdint>

extern "C" {
#include <libavutil/error.h>
}

// FrameSource is the one thing the media pipeline needs from its input.
//
// libavformat drives the pipeline: it asks for bytes when it is ready for them,
// so a live gRPC upload and a recorded archive appear here as the same shape.
// The contract is libavformat's own, and it is strict:
//
//   * one call returns exactly one RTP or RTCP packet, because packet
//     boundaries are what the demuxer reconstructs from the return value --
//     there is no length prefix and no RFC 4571 framing on this path;
//   * it blocks until there is one, and never returns 0. avio_alloc_context
//     documents that a stream protocol must return a proper AVERROR instead;
//   * it never returns AVERROR(EAGAIN). The AVIOContext latches that in its
//     error field and nothing in libavformat ever clears it, so one EAGAIN
//     wedges the demuxer for the life of the stream.
class FrameSource {
public:
    virtual ~FrameSource() = default;

    // Returns the number of bytes written into buf, or AVERROR_EOF once the
    // stream is over.
    virtual int next(uint8_t *buf, int cap) = 0;
};
