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
// The seam between the pipeline's stages.
//

#pragma once

extern "C" {
#include <libavformat/avformat.h>
}

#include "Uncopyable.hpp"

// PacketStage is one step of the media pipeline: access units in, access units
// on to whatever comes next.
//
// Only run_stream pulls; everything below it is pushed. A stage that has a
// downstream opens it from its own open(), passing the parameters and time base
// of what it will emit rather than what it received -- which is how a stage
// that rewrites the stream declares the change to the sink without run_stream
// having to know the difference.
//
// The stages of the remux chain are the policies the muxer should not own:
// KeyframeGate decides where a playable stream begins, DurationFiller supplies
// the timestamps the depacketiser leaves out. A transcode chain replaces both,
// because an encoder starts on a keyframe by construction and dates its own
// output.
class PacketStage : Uncopyable {
public:
    virtual ~PacketStage() = default;

    // Declares what will arrive. Returns 0, or a libav error code.
    [[nodiscard]] virtual int open(const AVCodecParameters *in,
                                   AVRational in_tb) = 0;

    // Takes one access unit. The packet is consumed either way: on success it
    // is held, forwarded or written, on failure it is released.
    [[nodiscard]] virtual int write(AVPacket *pkt) = 0;

    // Flushes what is held and closes. Idempotent.
    [[nodiscard]] virtual int finish() = 0;
};
