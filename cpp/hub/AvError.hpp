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
// libav error reporting.
//

#pragma once

#include <string>

extern "C" {
#include <libavutil/error.h>
}

// av_error renders a libav return code as text.
//
// It exists because every libav call reports failure as a negative int, and a
// bare number in a log or a gRPC status tells nobody anything. The previous
// revision of this pipeline used assert() instead, which reported nothing at
// all under NDEBUG.
inline std::string av_error(int rc) {
    char buf[AV_ERROR_MAX_STRING_SIZE] = {0};
    if (av_strerror(rc, buf, sizeof(buf)) < 0) {
        return "unknown libav error " + std::to_string(rc);
    }
    return {buf};
}
