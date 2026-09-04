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
// Where a playable stream begins.
//

#pragma once

#include <cstdint>

#include "PacketStage.hpp"

// KeyframeGate drops access units until the first keyframe.
//
// A live stream is joined wherever the camera happens to be, and a segment has
// to begin on a keyframe to be playable at all, so what arrives before the
// first one is undecodable. It is counted rather than stored, because a count
// that stays high says the camera's GOP is longer than anyone expected.
class KeyframeGate : public PacketStage {
public:
    KeyframeGate() = delete;

    explicit KeyframeGate(PacketStage &next);

    [[nodiscard]] int open(const AVCodecParameters *in,
                           AVRational in_tb) override;

    [[nodiscard]] int write(AVPacket *pkt) override;

    [[nodiscard]] int finish() override;

    // How many access units were dropped before the first keyframe.
    [[nodiscard]] uint64_t skipped() const { return skipped_; }

private:
    PacketStage &next_;
    bool saw_key_{false};
    uint64_t skipped_{0};
};
