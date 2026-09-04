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
// Where one camera's HLS output lives.
//

#include "StreamStorage.hpp"

#include <filesystem>
#include <fstream>
#include <set>
#include <string>
#include <system_error>

extern "C" {
#include <libavutil/error.h>
}

namespace {

// Deliberately a bare name; see StreamStorage::init_name.
constexpr const char *kInitName = "init.mp4";

// Beside the playlist rather than inside it: the muxer owns the playlist and
// rewrites it, so anything of ours kept there would not survive.
constexpr const char *kFingerprintName = "stream.params";

constexpr const char *kFragmentExtension = ".m4s";

} // namespace

bool stream_id_is_safe(const std::string &id) {
    if (id.empty() || id.size() > 128) {
        return false;
    }
    if (id == "." || id == "..") {
        return false;
    }
    for (const char c : id) {
        const bool ok = (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
                        (c >= '0' && c <= '9') || c == '.' || c == '-' || c == '_';
        if (!ok) {
            return false;
        }
    }
    return true;
}

StreamStorage::StreamStorage(std::string root, std::string user, std::string camera)
        : root_{std::move(root)}, user_{std::move(user)}, camera_{std::move(camera)} {}

int StreamStorage::prepare() {
    if (root_.empty() || !stream_id_is_safe(user_) || !stream_id_is_safe(camera_)) {
        return AVERROR(EINVAL);
    }

    const std::filesystem::path dir =
            std::filesystem::path(root_) / user_ / camera_;

    std::error_code ec;
    std::filesystem::create_directories(dir, ec);
    if (ec) {
        return AVERROR(ec.value() != 0 ? ec.value() : EIO);
    }

    dir_ = dir.string();
    return 0;
}

std::string StreamStorage::playlist() const {
    return (std::filesystem::path(dir_) / "stream.m3u8").string();
}

std::string StreamStorage::segments() const {
    return (std::filesystem::path(dir_) / "seg_%05d.m4s").string();
}

std::string StreamStorage::init() const {
    return (std::filesystem::path(dir_) / kInitName).string();
}

std::string StreamStorage::init_name() const { return kInitName; }

std::string StreamStorage::fingerprint_path() const {
    return (std::filesystem::path(dir_) / kFingerprintName).string();
}

void StreamStorage::reap_unlisted() const {
    // The playlist names its fragments one per line, every other line being a
    // tag. Anything on disk it does not name is unreachable.
    std::set<std::string> listed;
    {
        std::ifstream in(playlist());
        std::string line;
        while (std::getline(in, line)) {
            if (!line.empty() && line.back() == '\r') {
                line.pop_back();
            }
            if (!line.empty() && line.front() != '#') {
                listed.insert(line);
            }
        }
    }

    std::error_code ec;
    for (const auto &entry : std::filesystem::directory_iterator(dir_, ec)) {
        if (entry.path().extension() != kFragmentExtension) {
            continue;
        }
        if (listed.count(entry.path().filename().string()) != 0) {
            continue;
        }
        std::error_code rc;
        std::filesystem::remove(entry.path(), rc);
    }
}

int StreamStorage::reconcile(const std::string &want, Continuity *how) {
    if (dir_.empty()) {
        // prepare() has not run, so there is no directory to reconcile.
        return AVERROR(EINVAL);
    }

    std::string had;
    {
        std::ifstream in(fingerprint_path());
        std::getline(in, had);
    }

    if (had == want) {
        reap_unlisted();
        if (how != nullptr) {
            *how = Continuity::Append;
        }
        return 0;
    }

    // A different recording, or none. Everything in the directory described the
    // old one, including the initialisation segment.
    std::error_code ec;
    for (const auto &entry : std::filesystem::directory_iterator(dir_, ec)) {
        std::error_code rc;
        std::filesystem::remove_all(entry.path(), rc);
    }

    std::ofstream out(fingerprint_path(), std::ios::trunc);
    out << want << "\n";
    if (!out) {
        return AVERROR(EIO);
    }
    if (how != nullptr) {
        *how = Continuity::Fresh;
    }
    return 0;
}
