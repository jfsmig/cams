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
// One camera's stream, from RTP to HLS.
//

#pragma once

#include <cstdint>
#include <string>
#include <string_view>

#include "FrameSource.hpp"
#include "FragmentSink.hpp"
#include "StreamStorage.hpp"

// StreamStats is what a finished stream has to report.
struct StreamStats {
    // Access units written to a segment.
    uint64_t written{0};

    // Access units dropped for arriving before the first keyframe.
    uint64_t skipped{0};

    // Bytes of receiver report libavformat generated and the source discarded.
    uint64_t rtcp_discarded{0};

    // Access units dropped for carrying no presentation timestamp.
    uint64_t undated{0};

    // The track the description was reduced to, and the only one whose packets
    // belong on this pipeline.
    uint32_t track{0};

    // Whether this session added to a recording already in the directory, or
    // started one. See StreamStorage::reconcile.
    bool appended{false};
};

// run_stream turns one camera's RTP into HLS under storage, returning when the
// source is exhausted.
//
// sdp is the description as the camera published it; it is reduced here to the
// media the pipeline can carry. Returns 0 for a stream that ended cleanly, or a
// libav error code. stats may be null.
//
// It is shared by the gRPC service and the replay harness on purpose: a
// recorded capture then exercises the same code as a live camera, which is the
// only way the media path gets tested without one.
[[nodiscard]] int run_stream(std::string_view sdp, FrameSource &frames,
                             StreamStorage &storage, const HlsOptions &opts,
                             StreamStats *stats);

// track_of reports which media of sdp run_stream would carry, so that a caller
// can filter its source before handing it over. Returns false when there is no
// video.
[[nodiscard]] bool track_of(std::string_view sdp, uint32_t *track);
