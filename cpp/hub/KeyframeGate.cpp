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

#include "KeyframeGate.hpp"

KeyframeGate::KeyframeGate(PacketStage &next) : next_{next} {}

int KeyframeGate::open(const AVCodecParameters *in, AVRational in_tb) {
    // Nothing here rewrites the stream, so the downstream sees what arrived.
    return next_.open(in, in_tb);
}

int KeyframeGate::write(AVPacket *pkt) {
    if (!saw_key_) {
        if ((pkt->flags & AV_PKT_FLAG_KEY) == 0) {
            av_packet_unref(pkt);
            skipped_++;
            return 0;
        }
        saw_key_ = true;
    }
    return next_.write(pkt);
}

int KeyframeGate::finish() { return next_.finish(); }
