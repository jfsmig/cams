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

#include "Sdp.hpp"

#include <vector>

namespace {

// split_lines breaks a description into lines, dropping the terminators. SDP is
// defined with CRLF but cameras are not reliable about it, so both are accepted.
std::vector<std::string_view> split_lines(std::string_view sdp) {
    std::vector<std::string_view> out;
    while (!sdp.empty()) {
        const auto nl = sdp.find('\n');
        std::string_view line = (nl == std::string_view::npos) ? sdp : sdp.substr(0, nl);
        if (!line.empty() && line.back() == '\r') {
            line.remove_suffix(1);
        }
        out.push_back(line);
        if (nl == std::string_view::npos) {
            break;
        }
        sdp.remove_prefix(nl + 1);
    }
    return out;
}

bool starts_with(std::string_view s, std::string_view prefix) {
    return s.size() >= prefix.size() && s.substr(0, prefix.size()) == prefix;
}

} // namespace

VideoMedia keep_first_video_media(std::string_view sdp) {
    const auto lines = split_lines(sdp);

    // Everything up to the first "m=" describes the session and is kept as it
    // is; each "m=" opens a media section that runs to the next one.
    std::size_t first_media = lines.size();
    for (std::size_t i = 0; i < lines.size(); i++) {
        if (starts_with(lines[i], "m=")) {
            first_media = i;
            break;
        }
    }

    std::size_t begin = lines.size(), end = lines.size();
    uint32_t track = 0, seen = 0;
    for (std::size_t i = first_media; i < lines.size(); i++) {
        if (!starts_with(lines[i], "m=")) {
            continue;
        }
        if (begin != lines.size()) {
            // The section after the one we kept: that is where it ends.
            end = i;
            break;
        }
        if (starts_with(lines[i], "m=video")) {
            begin = i;
            track = seen;
        }
        seen++;
    }
    if (begin == lines.size()) {
        return {};
    }

    VideoMedia out;
    out.track = track;
    for (std::size_t i = 0; i < first_media; i++) {
        out.sdp.append(lines[i]).append("\r\n");
    }
    for (std::size_t i = begin; i < end; i++) {
        out.sdp.append(lines[i]).append("\r\n");
    }
    return out;
}
