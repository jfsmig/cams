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
// Session description handling.
//

#pragma once

#include <cstdint>
#include <string>
#include <string_view>

// VideoMedia is a session description reduced to one media, and which one.
struct VideoMedia {
    // The reduced description, or empty when the original described no video.
    std::string sdp;

    // The zero-based index of the "m=" section that was kept, which is the
    // track number the agent tags that media's packets with.
    uint32_t track{0};
};

// keep_first_video_media reduces a camera's session description to the session
// part plus its first video media.
//
// The hub needs this for two reasons. A camera describes every media it has --
// the reference capture offers H.264 video and AAC audio -- while the agent sets
// up and forwards only the video, so a description taken at face value leaves
// the demuxer with a stream that never receives a packet, and the muxer waiting
// on it to interleave. And with one media left, libavformat stops matching
// packets to streams by payload type, which is what lets the RTCP through
// instead of failing to attribute it.
VideoMedia keep_first_video_media(std::string_view sdp);
