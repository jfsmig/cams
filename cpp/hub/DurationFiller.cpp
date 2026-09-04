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

#include "DurationFiller.hpp"

DurationFiller::DurationFiller(PacketStage &next) : next_{next} {}

DurationFiller::~DurationFiller() {
    if (held_ != nullptr) {
        av_packet_free(&held_);
    }
}

int DurationFiller::open(const AVCodecParameters *in, AVRational in_tb) {
    // A duration is not a stream parameter, so the downstream sees what
    // arrived.
    return next_.open(in, in_tb);
}

int DurationFiller::write(AVPacket *pkt) {
    // An access unit with no presentation timestamp cannot be placed on a
    // timeline, and the muxer does not merely warn about it -- it refuses the
    // packet, which ends the stream.
    //
    // The first one always arrives that way: libavformat's depacketiser has no
    // reference yet to map the camera's RTP clock onto a presentation one, so
    // it hands over the whole opening unit -- a full IDR -- undated. Dropping
    // that would take the stream's first keyframe with it and leave the first
    // fragment beginning on a delta frame, which is undecodable however well
    // formed the rest is. The clock the depacketiser then unwraps starts at
    // zero, so zero is where the unit before the first dated one belongs.
    //
    // Undated in mid-stream is a different matter: there is nothing to derive
    // it from and it is dropped, and counted, because a count that climbs says
    // the camera is sending something this pipeline does not understand.
    if (pkt->pts == AV_NOPTS_VALUE) {
        if (held_ != nullptr) {
            av_packet_unref(pkt);
            undated_++;
            return 0;
        }
        pkt->pts = 0;
        pkt->dts = 0;
    }

    if (held_ == nullptr) {
        held_ = av_packet_alloc();
        if (held_ == nullptr) {
            av_packet_unref(pkt);
            return AVERROR(ENOMEM);
        }
        av_packet_move_ref(held_, pkt);
        return 0;
    }

    if (pkt->pts != AV_NOPTS_VALUE && held_->pts != AV_NOPTS_VALUE &&
        pkt->pts > held_->pts) {
        held_->duration = pkt->pts - held_->pts;
        last_duration_ = held_->duration;
    }

    const int rc = next_.write(held_);
    av_packet_unref(held_);
    av_packet_move_ref(held_, pkt);
    return rc;
}

int DurationFiller::finish() {
    int rc = 0;
    if (held_ != nullptr) {
        // Nothing follows the last packet, so it inherits the previous
        // interval. Overwritten rather than defaulted: libavformat hands the
        // final access unit a duration of one tick -- eleven microseconds at
        // 90 kHz -- which is not a plausible frame and is worse than the
        // interval every packet before it measured.
        if (last_duration_ > 0) {
            held_->duration = last_duration_;
        }
        rc = next_.write(held_);
        av_packet_free(&held_);
    }
    const int frc = next_.finish();
    return rc < 0 ? rc : frc;
}
