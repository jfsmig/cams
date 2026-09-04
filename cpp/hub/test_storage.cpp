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
// What survives from one session to the next.
//
// reconcile is the whole of the reconnect story and it is pure filesystem, so
// it is driven here directly rather than through a replay.
//

#include <cstdarg>
#include <cstdio>
#include <filesystem>
#include <fstream>
#include <set>
#include <string>
#include <vector>
#include <unistd.h>

#include "StreamStorage.hpp"

namespace {

int failures = 0;

void fail(const char *file, int line, const char *expr, const char *fmt, ...) {
    fprintf(stderr, "FAIL %s:%d: %s\n  ", file, line, expr);
    va_list ap;
    va_start(ap, fmt);
    vfprintf(stderr, fmt, ap);
    va_end(ap);
    fputc('\n', stderr);
    failures++;
}

#define CHECK(cond, ...)                                                       \
    do {                                                                       \
        if (!(cond)) {                                                         \
            fail(__FILE__, __LINE__, #cond, __VA_ARGS__);                      \
        }                                                                      \
    } while (0)

std::filesystem::path fresh_root(const char *name) {
    const auto root = std::filesystem::temp_directory_path() /
                      ("cams-storage-" + std::to_string(::getpid()) + "-" + name);
    std::filesystem::remove_all(root);
    return root;
}

void touch(const std::filesystem::path &p, const char *body = "x") {
    std::ofstream out(p, std::ios::trunc);
    out << body;
}

std::set<std::string> names_in(const std::string &dir) {
    std::set<std::string> out;
    for (const auto &e : std::filesystem::directory_iterator(dir)) {
        out.insert(e.path().filename().string());
    }
    return out;
}

// A playlist naming exactly the fragments given.
void write_playlist(const StreamStorage &s,
                    const std::vector<std::string> &fragments) {
    std::ofstream out(s.playlist(), std::ios::trunc);
    out << "#EXTM3U\n#EXT-X-VERSION:7\n";
    out << "#EXT-X-MAP:URI=\"" << s.init_name() << "\"\n";
    for (const auto &f : fragments) {
        out << "#EXTINF:4.000,\n" << f << "\n";
    }
    out << "#EXT-X-ENDLIST\n";
}

void an_empty_directory_starts_a_recording() {
    const auto root = fresh_root("empty");
    StreamStorage s(root.string(), "u", "c");
    CHECK(s.prepare() == 0, "prepare failed");

    auto how = StreamStorage::Continuity::Append;
    CHECK(s.reconcile("h264 100x100 4:dead", &how) == 0, "reconcile failed");
    CHECK(how == StreamStorage::Continuity::Fresh,
          "an empty directory reported Append");
}

void the_same_recording_is_continued() {
    const auto root = fresh_root("same");
    StreamStorage s(root.string(), "u", "c");
    CHECK(s.prepare() == 0, "prepare failed");

    auto how = StreamStorage::Continuity::Append;
    CHECK(s.reconcile("h264 100x100 4:dead", &how) == 0, "first reconcile failed");
    CHECK(how == StreamStorage::Continuity::Fresh, "first pass was not Fresh");

    // What the muxer would have left behind.
    touch(s.init());
    write_playlist(s, {"seg_00000.m4s"});
    touch(std::filesystem::path(s.directory()) / "seg_00000.m4s");

    CHECK(s.reconcile("h264 100x100 4:dead", &how) == 0, "second reconcile failed");
    CHECK(how == StreamStorage::Continuity::Append,
          "the same recording was not continued");

    const auto left = names_in(s.directory());
    CHECK(left.count("seg_00000.m4s") == 1,
          "a listed fragment was reaped");
    CHECK(left.count("init.mp4") == 1, "the initialisation segment was reaped");
}

// The reason reconcile exists at all: the muxer cannot see fragments that fell
// off a playlist it had not read, so each reconnect would orphan one.
void unlisted_fragments_are_reaped() {
    const auto root = fresh_root("reap");
    StreamStorage s(root.string(), "u", "c");
    CHECK(s.prepare() == 0, "prepare failed");

    auto how = StreamStorage::Continuity::Append;
    CHECK(s.reconcile("h264 100x100 4:dead", &how) == 0, "reconcile failed");

    touch(s.init());
    const auto dir = std::filesystem::path(s.directory());
    touch(dir / "seg_00000.m4s");
    touch(dir / "seg_00001.m4s");
    touch(dir / "seg_00002.m4s");
    // Only the middle one is still listed.
    write_playlist(s, {"seg_00001.m4s"});

    CHECK(s.reconcile("h264 100x100 4:dead", &how) == 0, "reconcile failed");
    CHECK(how == StreamStorage::Continuity::Append, "not Append");

    const auto left = names_in(s.directory());
    CHECK(left.count("seg_00001.m4s") == 1, "the listed fragment was reaped");
    CHECK(left.count("seg_00000.m4s") == 0, "an unlisted fragment survived");
    CHECK(left.count("seg_00002.m4s") == 0, "an unlisted fragment survived");
    CHECK(left.count("init.mp4") == 1, "the initialisation segment was reaped");
    CHECK(left.count("stream.params") == 1, "the fingerprint was reaped");
}

// A different codec, or different parameter sets, means the initialisation
// segment no longer describes what is already there.
void a_different_recording_replaces_the_old_one() {
    const auto root = fresh_root("differs");
    StreamStorage s(root.string(), "u", "c");
    CHECK(s.prepare() == 0, "prepare failed");

    auto how = StreamStorage::Continuity::Append;
    CHECK(s.reconcile("h264 2560x1920 22:aaaa", &how) == 0, "reconcile failed");
    touch(s.init());
    write_playlist(s, {"seg_00000.m4s"});
    touch(std::filesystem::path(s.directory()) / "seg_00000.m4s");

    // Same dimensions, different parameter sets: still a different recording.
    CHECK(s.reconcile("h264 2560x1920 22:bbbb", &how) == 0, "reconcile failed");
    CHECK(how == StreamStorage::Continuity::Fresh,
          "a changed SPS was treated as the same recording");

    const auto left = names_in(s.directory());
    CHECK(left.count("seg_00000.m4s") == 0,
          "a fragment of the old recording survived");
    CHECK(left.count("init.mp4") == 0,
          "the old initialisation segment survived");
    CHECK(left.count("stream.params") == 1,
          "the new fingerprint was not written");

    // And it is now the recording of record.
    CHECK(s.reconcile("h264 2560x1920 22:bbbb", &how) == 0, "reconcile failed");
    CHECK(how == StreamStorage::Continuity::Append,
          "the replacement recording was not remembered");
}

void a_codec_change_replaces_the_old_one() {
    const auto root = fresh_root("codec");
    StreamStorage s(root.string(), "u", "c");
    CHECK(s.prepare() == 0, "prepare failed");

    auto how = StreamStorage::Continuity::Append;
    CHECK(s.reconcile("h264 1920x1080 22:aaaa", &how) == 0, "reconcile failed");
    touch(std::filesystem::path(s.directory()) / "seg_00000.m4s");

    CHECK(s.reconcile("hevc 1920x1080 22:aaaa", &how) == 0, "reconcile failed");
    CHECK(how == StreamStorage::Continuity::Fresh,
          "h264 and hevc were treated as one recording");
    CHECK(names_in(s.directory()).count("seg_00000.m4s") == 0,
          "an h264 fragment survived a switch to hevc");
}

void reconcile_without_prepare_is_refused() {
    StreamStorage s("/nonexistent-root", "u", "c");
    auto how = StreamStorage::Continuity::Append;
    // prepare() was not called, so there is no directory.
    CHECK(s.reconcile("h264 1x1 0:0", &how) < 0,
          "reconcile succeeded with no directory");
}

} // namespace

int main() {
    an_empty_directory_starts_a_recording();
    the_same_recording_is_continued();
    unlisted_fragments_are_reaped();
    a_different_recording_replaces_the_old_one();
    a_codec_change_replaces_the_old_one();
    reconcile_without_prepare_is_refused();

    if (failures == 0) {
        fprintf(stderr, "PASS\n");
        return 0;
    }
    fprintf(stderr, "%d check(s) failed\n", failures);
    return 1;
}
