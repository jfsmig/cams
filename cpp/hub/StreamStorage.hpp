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

#pragma once

#include <string>

#include "Uncopyable.hpp"

// StreamStorage owns one (user, camera) stream's directory: the names of the
// files in it, and what survives from one session to the next.
//
// It hands the muxer paths rather than bytes, because the HLS muxer is declared
// AVFMT_NOFILE and opens every playlist and fragment itself.
class StreamStorage : Uncopyable {
public:
    StreamStorage() = delete;

    StreamStorage(std::string root, std::string user, std::string camera);

    ~StreamStorage() = default;

    // Validates the identifiers and creates the directory.
    //
    // The identifiers arrive in gRPC metadata, which is to say from a peer, and
    // they become path components -- so they are checked here rather than
    // trusted. Returns 0, or a libav error code.
    [[nodiscard]] int prepare();

    [[nodiscard]] const std::string &directory() const { return dir_; }

    // The playlist the muxer writes and a player fetches.
    [[nodiscard]] std::string playlist() const;

    // The pattern the muxer names fragments after.
    [[nodiscard]] std::string segments() const;

    // The initialisation segment. fMP4 carries the sample description once,
    // here, rather than in every fragment; a player fetches it first and the
    // playlist points at it with EXT-X-MAP.
    [[nodiscard]] std::string init() const;

    // Continuity says whether a session may add to what is already here.
    enum class Continuity {
        // The directory held a different recording, or nothing. It has been
        // emptied and this session starts the playlist.
        Fresh,
        // The directory holds the same recording, so this session adds to it.
        Append,
    };

    // Settles whether this session continues the last one, and leaves the
    // directory fit for it either way.
    //
    // want describes what is about to be recorded -- codec, dimensions and
    // parameter sets. Two sessions may only share a playlist if it matches:
    // fMP4 carries the sample description once, in the initialisation segment,
    // and that file is rewritten per session. Appending fragments described by
    // a different one would leave the older fragments referenced by an
    // EXT-X-MAP that does not describe them, which no player can detect and
    // none can play.
    //
    // On a match, the fragments the playlist no longer lists are reaped: the
    // muxer cannot see them -- they fell off a playlist it had not read yet --
    // so it would never reclaim them, and one would be orphaned per reconnect.
    // On a mismatch the directory is emptied, because a recording that cannot
    // be continued is worth less than the one replacing it.
    //
    // Returns 0, or a libav error code.
    [[nodiscard]] int reconcile(const std::string &want, Continuity *how);

    // The same file as init(), named the way the muxer has to be told about it.
    // The HLS muxer joins the playlist's own directory with whatever it is
    // given here, so an absolute path becomes that directory twice over and
    // the open fails; and this is also the URI that lands in EXT-X-MAP, where
    // a filesystem path would be wrong for a player anyway.
    [[nodiscard]] std::string init_name() const;

private:
    // Where the description of the current recording is kept, so that the next
    // session can tell whether it is the same one.
    [[nodiscard]] std::string fingerprint_path() const;

    // Removes every fragment the playlist does not name. Missing playlist means
    // nothing is named, so everything goes.
    void reap_unlisted() const;

    std::string root_;
    std::string user_;
    std::string camera_;
    std::string dir_;
};

// stream_id_is_safe reports whether an identifier may be used as a path
// component: non-empty, short enough, made only of characters that cannot
// escape a directory, and not one of the relative names.
[[nodiscard]] bool stream_id_is_safe(const std::string &id);
