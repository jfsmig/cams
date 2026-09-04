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
// The timestamps the depacketiser leaves out.
//

#pragma once

#include <cstdint>

#include "PacketStage.hpp"

// DurationFiller holds one access unit back so that its successor can date it.
//
// The RTP depacketiser leaves duration at zero. Left alone, the HLS muxer then
// estimates every #EXTINF from timestamp deltas and warns about it once per
// packet -- the same arithmetic, done noisily. Holding one packet costs a frame
// of latency and does it quietly.
//
// The last access unit has no successor to date it, so it inherits the previous
// interval instead of the one tick libavformat gives it. Letting the muxer
// guess costs one frame off the final #EXTINF, and #EXTINF is what a player
// builds its timeline from: under-reporting accumulates into drift between
// playlist time and media time, which is what breaks a seek by
// EXT-X-PROGRAM-DATE-TIME.
class DurationFiller : public PacketStage {
public:
    DurationFiller() = delete;

    explicit DurationFiller(PacketStage &next);

    ~DurationFiller() override;

    [[nodiscard]] int open(const AVCodecParameters *in,
                           AVRational in_tb) override;

    [[nodiscard]] int write(AVPacket *pkt) override;

    [[nodiscard]] int finish() override;

    // How many access units arrived mid-stream with no presentation timestamp
    // and were dropped. The opening one is dated rather than dropped and does
    // not count here; see write().
    [[nodiscard]] uint64_t undated() const { return undated_; }

private:
    PacketStage &next_;
    AVPacket *held_{nullptr};
    int64_t last_duration_{0};
    uint64_t undated_{0};
};
