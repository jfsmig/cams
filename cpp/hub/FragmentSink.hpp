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
// The HLS side of the pipeline: fragments out, and the playlist naming them.
//

#pragma once

#include <cstdint>

extern "C" {
#include <libavformat/avformat.h>
}

#include "PacketStage.hpp"
#include "StreamStorage.hpp"

// HlsOptions is what a caller chooses about the output.
struct HlsOptions {
    // Target length of a fragment. Fragments only ever break on a keyframe, so
    // a camera whose GOP is longer than this produces longer ones, and this is
    // a floor rather than a length.
    int segment_seconds{4};

    // How far back the playlist reaches, which is also what bounds the
    // directory. One number for both on purpose: what a viewer can seek to and
    // what still exists have to be the same set, or the playlist names
    // fragments that were deleted. Expressed in seconds because that is what a
    // retention policy is about; the muxer's count is derived from it.
    //
    // The directory holds exactly one fragment more than the playlist names,
    // for the benefit of a player still fetching the one that just fell off.
    int retention_seconds{1800};

    // Keep every fragment ever written, and list them all. The replay harness
    // needs this because it asserts on the output. A live stream must not have
    // it: nothing else bounds the directory.
    bool keep_all{false};
};

// FragmentSink is the pipeline's terminal stage: it declares the output track
// and writes access units to it. Nothing else -- where a stream may start and
// what dates an access unit are decided upstream, by KeyframeGate and
// DurationFiller.
//
// It remuxes and never decodes, and carries whatever the source depacketised:
// H.264 and H.265 both arrive as access units in Annex-B, and libavformat's mov
// muxer converts those to the length-prefixed form MP4 wants on its own,
// writing avcC or hvcC from the Annex-B parameter sets. The *_mp4toannexb
// filters would be the wrong direction.
//
// Fragmented MP4 rather than MPEG-TS, because it is the only shape the rest of
// the delivery path can use: LL-HLS parts and Media Source Extensions both
// require fMP4, and MPEG-TS is a dead end for both. The cost is that an fMP4
// track cannot be declared without the picture dimensions -- libavformat
// refuses one with "dimensions not set" -- which is why RtpSource::open settles
// them before this stage is opened.
class FragmentSink : public PacketStage {
public:
    FragmentSink() = delete;

    // how comes from StreamStorage::reconcile: appending adds to the playlist
    // already in the directory instead of truncating it.
    FragmentSink(const StreamStorage &storage, HlsOptions opts,
                 StreamStorage::Continuity how);

    ~FragmentSink() override;

    // Declares the output track from the parameters the stage above emits, and
    // writes the playlist header. Returns 0, or a libav error code.
    [[nodiscard]] int open(const AVCodecParameters *in,
                           AVRational in_tb) override;

    // Writes one access unit. The packet is consumed either way.
    [[nodiscard]] int write(AVPacket *pkt) override;

    // Closes the playlist. Idempotent.
    [[nodiscard]] int finish() override;

    [[nodiscard]] uint64_t written() const { return written_; }

private:
    const StreamStorage &storage_;
    HlsOptions opts_;
    StreamStorage::Continuity how_;

    AVFormatContext *ctx_{nullptr};
    AVRational in_tb_{0, 1};
    bool opened_{false};
    uint64_t written_{0};
};
