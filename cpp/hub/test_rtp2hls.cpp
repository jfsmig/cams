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
// Replays a recorded camera stream through the media pipeline and checks that
// it comes out as HLS.
//
// The capture is what cams-cli's tar sink writes: one .sdp entry, then a .rtp or
// .rtcp entry per packet, in the order the camera sent them. Replaying it
// exercises the same run_stream that a live upload does, which is the only way
// this pipeline is tested without a camera on the bench.
//

#include <cstdio>
#include <cstdlib>
#include <cstring>
#include <filesystem>
#include <fstream>
#include <string>
#include <vector>

#include <archive.h>
#include <archive_entry.h>

#include <unistd.h>

#include "AvError.hpp"
#include "FrameSource.hpp"
#include "MediaPipeline.hpp"
#include "Sdp.hpp"
#include "StreamStorage.hpp"

namespace {

int failures = 0;

// check reports every failure instead of stopping at the first, and unlike the
// assert() this file used to rely on it is still there under NDEBUG.
#define CHECK(Cond, ...)                                                    \
    do {                                                                    \
        if (!(Cond)) {                                                      \
            fprintf(stderr, "FAIL %s:%d: %s\n  ", __FILE__, __LINE__, #Cond); \
            fprintf(stderr, __VA_ARGS__);                                   \
            fprintf(stderr, "\n");                                          \
            failures++;                                                     \
        }                                                                   \
    } while (0)

bool ends_with(const std::string &s, const std::string &suffix) {
    return s.size() >= suffix.size() &&
           0 == s.compare(s.size() - suffix.size(), suffix.size(), suffix);
}

// Packet is one entry of the capture, held in memory: the archive is read once
// and replayed from there, because libarchive cannot be asked for the next
// entry from inside a read callback that libavformat drives.
struct Packet {
    std::string name;
    std::vector<uint8_t> body;
};

struct Capture {
    std::string sdp;
    std::vector<Packet> packets;
};

// read_capture loads the whole archive: the description and every RTP and RTCP
// entry, in order.
bool read_capture(const char *path, Capture *out) {
    auto *handle = archive_read_new();
    if (handle == nullptr) {
        fprintf(stderr, "archive_read_new failed\n");
        return false;
    }
    bool ok = true;
    if (archive_read_support_filter_none(handle) != ARCHIVE_OK ||
        archive_read_support_format_tar(handle) != ARCHIVE_OK ||
        archive_read_open_filename(handle, path, 10240) != ARCHIVE_OK) {
        fprintf(stderr, "cannot open %s: %s\n", path, archive_error_string(handle));
        archive_read_free(handle);
        return false;
    }

    struct archive_entry *entry = nullptr;
    while (archive_read_next_header(handle, &entry) == ARCHIVE_OK) {
        const std::string name(archive_entry_pathname(entry));
        const auto size = static_cast<size_t>(archive_entry_size(entry));

        const bool is_sdp = ends_with(name, ".sdp");
        const bool is_rtp = ends_with(name, ".rtp") || ends_with(name, ".rtcp");
        if (!is_sdp && !is_rtp) {
            continue;
        }

        std::vector<uint8_t> body(size);
        if (size > 0) {
            const auto n = archive_read_data(handle, body.data(), size);
            if (n < 0 || static_cast<size_t>(n) != size) {
                fprintf(stderr, "short read on %s: %s\n", name.c_str(),
                        archive_error_string(handle));
                ok = false;
                break;
            }
        }

        if (is_sdp) {
            if (out->sdp.empty()) {
                out->sdp.assign(reinterpret_cast<const char *>(body.data()), size);
            }
        } else {
            out->packets.push_back({name, std::move(body)});
        }
    }

    archive_read_free(handle);
    return ok;
}

// ReplayFrameSource hands the captured packets to libavformat one at a time,
// honouring the FrameSource contract: exactly one packet per call, and
// AVERROR_EOF only when there are none left.
class ReplayFrameSource : public FrameSource {
public:
    explicit ReplayFrameSource(const std::vector<Packet> &packets) : packets_{packets} {}

    int next(uint8_t *buf, int cap) override {
        while (at_ < packets_.size()) {
            const Packet &p = packets_[at_++];
            if (p.body.empty() || p.body.size() > static_cast<size_t>(cap)) {
                skipped_++;
                continue;
            }
            memcpy(buf, p.body.data(), p.body.size());
            return static_cast<int>(p.body.size());
        }
        return AVERROR_EOF;
    }

    [[nodiscard]] size_t delivered() const { return at_ - skipped_; }

    [[nodiscard]] size_t skipped() const { return skipped_; }

private:
    const std::vector<Packet> &packets_;
    size_t at_{0};
    size_t skipped_{0};
};

std::string read_file(const std::filesystem::path &p) {
    std::ifstream in(p, std::ios::binary);
    return {std::istreambuf_iterator<char>(in), std::istreambuf_iterator<char>()};
}

// Playback is what the playlist is for, so the test reads its own output back
// with libavformat rather than only inspecting the text. This is the check that
// would catch a segment full of bytes no decoder can use.
struct Playback {
    int streams{0};
    AVCodecID codec{AV_CODEC_ID_NONE};
    int width{0};
    int height{0};
    int64_t packets{0};
    int64_t keyframes{0};
    double seconds{0};
    bool timestamps_increase{true};
};

int verify_playback(const std::string &playlist, Playback *out) {
    AVFormatContext *ic = nullptr;
    int rc = avformat_open_input(&ic, playlist.c_str(), nullptr, nullptr);
    if (rc < 0) {
        return rc;
    }
    rc = avformat_find_stream_info(ic, nullptr);
    if (rc < 0) {
        avformat_close_input(&ic);
        return rc;
    }

    out->streams = static_cast<int>(ic->nb_streams);
    if (ic->nb_streams > 0) {
        const AVCodecParameters *par = ic->streams[0]->codecpar;
        out->codec = par->codec_id;
        out->width = par->width;
        out->height = par->height;
    }

    AVPacket *pkt = av_packet_alloc();
    if (pkt == nullptr) {
        avformat_close_input(&ic);
        return AVERROR(ENOMEM);
    }
    int64_t first = AV_NOPTS_VALUE, last = AV_NOPTS_VALUE, previous = AV_NOPTS_VALUE;
    while (av_read_frame(ic, pkt) >= 0) {
        out->packets++;
        if ((pkt->flags & AV_PKT_FLAG_KEY) != 0) {
            out->keyframes++;
        }
        if (pkt->pts != AV_NOPTS_VALUE) {
            if (first == AV_NOPTS_VALUE) {
                first = pkt->pts;
            }
            if (previous != AV_NOPTS_VALUE && pkt->pts < previous) {
                out->timestamps_increase = false;
            }
            previous = pkt->pts;
            last = pkt->pts;
        }
        av_packet_unref(pkt);
    }
    av_packet_free(&pkt);

    if (first != AV_NOPTS_VALUE && last != AV_NOPTS_VALUE && ic->nb_streams > 0) {
        out->seconds = static_cast<double>(last - first) *
                       av_q2d(ic->streams[0]->time_base);
    }
    avformat_close_input(&ic);
    return 0;
}

size_t count_occurrences(const std::string &haystack, const std::string &needle) {
    size_t n = 0;
    for (size_t at = haystack.find(needle); at != std::string::npos;
         at = haystack.find(needle, at + needle.size())) {
        n++;
    }
    return n;
}

} // namespace

int main(int argc, char **argv) {
    if (argc != 2) {
        fprintf(stderr, "usage: %s PATH-TO-CAPTURE.tar\n", argv[0]);
        return 2;
    }
    const char *archive_path = argv[1];

    av_log_set_level(AV_LOG_ERROR);

    Capture capture;
    if (!read_capture(archive_path, &capture)) {
        return 1;
    }
    fprintf(stderr, "replaying %s: %zu packets\n", archive_path, capture.packets.size());

    CHECK(!capture.sdp.empty(), "the capture holds no session description");
    CHECK(!capture.packets.empty(), "the capture holds no packets");
    if (failures != 0) {
        return 1;
    }

    // The description has to reduce to exactly one video media, which is what
    // the pipeline can carry. The reference capture offers video and audio.
    const VideoMedia media = keep_first_video_media(capture.sdp);
    CHECK(!media.sdp.empty(), "no video media in the description");
    CHECK(media.sdp.find("m=audio") == std::string::npos,
          "the audio media survived the filter");
    CHECK(media.sdp.find("m=video") != std::string::npos,
          "the video media did not survive the filter");
    CHECK(media.sdp.find("H264/90000") != std::string::npos,
          "the video format was lost with the filter");

    // A directory of its own per run, so a stale playlist cannot pass for a
    // fresh one.
    const auto root = std::filesystem::temp_directory_path() /
                      ("cams-replay-" + std::to_string(::getpid()));
    std::filesystem::remove_all(root);

    StreamStorage storage(root.string(), "replay", "camera");
    // Every fragment is kept: this asserts on the output, and a rolling window
    // would delete the evidence. The retention is moot under keep_all.
    const HlsOptions opts{4, 3600, true};

    ReplayFrameSource frames(capture.packets);
    StreamStats stats;

    const int rc = run_stream(capture.sdp, frames, storage, opts, &stats);
    CHECK(rc == 0, "run_stream: %s", av_error(rc).c_str());

    fprintf(stderr,
            "delivered=%zu skipped_source=%zu written=%llu skipped_prekey=%llu"
            " undated=%llu rtcp_discarded=%llu track=%u\n",
            frames.delivered(), frames.skipped(),
            static_cast<unsigned long long>(stats.written),
            static_cast<unsigned long long>(stats.skipped),
            static_cast<unsigned long long>(stats.undated),
            static_cast<unsigned long long>(stats.rtcp_discarded),
            stats.track);

    // Every packet of the capture is replayed. The cap of eleven that used to
    // be here stopped before the first access unit was even complete.
    CHECK(frames.delivered() + frames.skipped() == capture.packets.size(),
          "replayed %zu of %zu packets", frames.delivered() + frames.skipped(),
          capture.packets.size());
    CHECK(stats.written > 0, "no access unit reached a segment");

    // What the output has to look like for a player to be able to use it.
    const auto playlist_path = std::filesystem::path(storage.playlist());
    CHECK(std::filesystem::exists(playlist_path), "no playlist at %s",
          playlist_path.string().c_str());

    const std::string playlist = read_file(playlist_path);
    CHECK(playlist.rfind("#EXTM3U", 0) == 0, "the playlist does not open with #EXTM3U");
    CHECK(playlist.find("#EXT-X-ENDLIST") != std::string::npos,
          "the playlist has no end marker, so the trailer was not written");

    // fMP4 carries the sample description once, in the initialisation segment,
    // so a fragment is undecodable without it and a playlist that does not
    // point at it is unusable.
    const auto init_path = std::filesystem::path(storage.init());
    CHECK(std::filesystem::exists(init_path), "no initialisation segment at %s",
          init_path.string().c_str());
    if (std::filesystem::exists(init_path)) {
        CHECK(std::filesystem::file_size(init_path) > 0,
              "the initialisation segment is empty");
    }
    CHECK(playlist.find("#EXT-X-MAP:") != std::string::npos,
          "the playlist has no EXT-X-MAP, so a player cannot find the "
          "initialisation segment");
    // The URI has to be resolvable against the playlist, which a filesystem
    // path would not be.
    CHECK(playlist.find("#EXT-X-MAP:URI=\"" + storage.init_name() + "\"") !=
                  std::string::npos,
          "EXT-X-MAP does not name %s relative to the playlist",
          storage.init_name().c_str());

    // Seeking to the moment an event was reported is what the stored fragments
    // are for, and this tag is what makes it possible.
    CHECK(playlist.find("#EXT-X-PROGRAM-DATE-TIME:") != std::string::npos,
          "the playlist carries no EXT-X-PROGRAM-DATE-TIME, so it cannot be "
          "seeked by wall clock");

    const size_t segments = count_occurrences(playlist, "#EXTINF:");
    CHECK(segments > 0, "the playlist lists no segment");

    // Each listed segment has to exist and hold something.
    size_t on_disk = 0;
    double total_seconds = 0;
    if (std::filesystem::exists(storage.directory())) {
        for (const auto &e : std::filesystem::directory_iterator(storage.directory())) {
            if (e.path().extension() == ".m4s") {
                on_disk++;
                CHECK(e.file_size() > 0, "segment %s is empty",
                      e.path().filename().string().c_str());
            }
        }
    }
    CHECK(on_disk == segments, "the playlist lists %zu segments and %zu are on disk",
          segments, on_disk);

    // The durations have to add up to something: a playlist of zero-length
    // segments would satisfy every check above.
    for (size_t at = playlist.find("#EXTINF:"); at != std::string::npos;
         at = playlist.find("#EXTINF:", at + 1)) {
        total_seconds += atof(playlist.c_str() + at + strlen("#EXTINF:"));
    }
    CHECK(total_seconds > 0.0, "the playlist covers no time at all");

    fprintf(stderr, "playlist: %zu segments, %.3f seconds, under %s\n",
            segments, total_seconds, storage.directory().c_str());

    // Read the whole thing back. Everything above says the playlist is
    // well-formed; this says a player can use it.
    Playback back;
    const int prc = verify_playback(storage.playlist(), &back);
    CHECK(prc == 0, "reading the playlist back: %s", av_error(prc).c_str());
    if (prc == 0) {
        fprintf(stderr,
                "playback: %d stream(s), %s %dx%d, %lld packets, %lld keyframes,"
                " %.3f seconds\n",
                back.streams, avcodec_get_name(back.codec), back.width, back.height,
                static_cast<long long>(back.packets),
                static_cast<long long>(back.keyframes), back.seconds);

        CHECK(back.streams == 1, "the output holds %d streams, want 1", back.streams);
        CHECK(back.codec == AV_CODEC_ID_H264, "the output codec is %s, want h264",
              avcodec_get_name(back.codec));
        // The description does not carry the picture size; RtpSource::open
        // settles it, because an fMP4 track cannot be declared without it.
        // Zero here would mean the parameter sets never reached the fragments.
        CHECK(back.width > 0 && back.height > 0,
              "the output reports %dx%d, so the parameter sets did not survive",
              back.width, back.height);
        CHECK(back.packets > 0, "the output yielded no packet");
        CHECK(back.keyframes > 0, "the output holds no keyframe, so it cannot be joined");
        CHECK(back.timestamps_increase, "the output timestamps do not increase");
        // Muxing must not lose access units.
        CHECK(back.packets == static_cast<int64_t>(stats.written),
              "wrote %llu access units and read back %lld",
              static_cast<unsigned long long>(stats.written),
              static_cast<long long>(back.packets));
    }

    // A second pass with a bounded retention. The window handed to the muxer is
    // derived from retention_seconds / segment_seconds, and what falls off it
    // has to be deleted rather than merely unlisted: a playlist naming a
    // fragment that is gone is exactly the failure this guards against.
    {
        StreamStorage rstorage((root / "retained").string(), "replay", "camera");
        // One-second fragments and a two-second retention, so the capture
        // outruns the window and something has to be dropped.
        const HlsOptions ropts{1, 2, false};
        ReplayFrameSource rframes(capture.packets);
        StreamStats rstats;

        const int rrc = run_stream(capture.sdp, rframes, rstorage, ropts, &rstats);
        CHECK(rrc == 0, "run_stream (retained): %s", av_error(rrc).c_str());

        // The same replay with nothing dropped, as the baseline to compare
        // against: without it, a retention that kept everything would pass.
        StreamStorage ustorage((root / "unbounded").string(), "replay", "camera");
        const HlsOptions uopts{1, 3600, true};
        ReplayFrameSource uframes(capture.packets);
        const int urc = run_stream(capture.sdp, uframes, ustorage, uopts, nullptr);
        CHECK(urc == 0, "run_stream (unbounded): %s", av_error(urc).c_str());
        const size_t segments_all =
                urc == 0 ? count_occurrences(
                                   read_file(std::filesystem::path(ustorage.playlist())),
                                   "#EXTINF:")
                         : 0;

        if (rrc == 0) {
            const std::string rplaylist =
                    read_file(std::filesystem::path(rstorage.playlist()));
            const size_t listed = count_occurrences(rplaylist, "#EXTINF:");
            size_t on_disk_now = 0;
            for (const auto &e :
                 std::filesystem::directory_iterator(rstorage.directory())) {
                if (e.path().extension() == ".m4s") {
                    on_disk_now++;
                }
            }
            fprintf(stderr,
                    "retention: %llu written, %zu listed, %zu on disk\n",
                    static_cast<unsigned long long>(rstats.written), listed,
                    on_disk_now);

            CHECK(listed > 0, "the retained playlist lists no fragment");
            // Every listed fragment must exist, and at most one unlisted one
            // may survive it -- hls_delete_threshold. More than that means
            // retention is bounding the playlist and not the directory.
            CHECK(on_disk_now >= listed,
                  "the retained playlist lists %zu fragments and only %zu are "
                  "on disk",
                  listed, on_disk_now);
            CHECK(on_disk_now <= listed + 1,
                  "%zu fragments on disk for a playlist of %zu, so nothing is "
                  "being deleted",
                  on_disk_now, listed);
            CHECK(listed < segments_all,
                  "the retention dropped nothing: %zu listed of %zu written "
                  "unbounded",
                  listed, segments_all);
        }
    }

    // A reconnect must extend the recording, not overwrite it. Every RTSP
    // reconnect -- a camera reboot, a network blip, the agent's own retry loop
    // -- opens a fresh upload against the same directory, and the muxer used to
    // truncate the playlist and restart fragment numbering at zero.
    {
        StreamStorage rstorage((root / "reconnect").string(), "replay", "camera");
        const HlsOptions ropts{4, 3600, true};

        size_t listed_before = 0;
        for (int attempt = 1; attempt <= 3; attempt++) {
            ReplayFrameSource frames_again(capture.packets);
            StreamStats again;
            const int arc = run_stream(capture.sdp, frames_again, rstorage, ropts,
                                       &again);
            CHECK(arc == 0, "run_stream (reconnect %d): %s", attempt,
                  av_error(arc).c_str());
            if (arc != 0) {
                break;
            }

            const size_t listed = count_occurrences(
                    read_file(std::filesystem::path(rstorage.playlist())),
                    "#EXTINF:");
            fprintf(stderr, "reconnect %d: appended=%d listed=%zu\n", attempt,
                    again.appended ? 1 : 0, listed);

            // The first pass has nothing to add to; every later one does.
            CHECK(again.appended == (attempt > 1),
                  "attempt %d reported appended=%d", attempt,
                  again.appended ? 1 : 0);
            CHECK(listed > listed_before,
                  "attempt %d left %zu fragments listed, no more than the %zu "
                  "before it, so the recording was overwritten",
                  attempt, listed, listed_before);
            listed_before = listed;
        }

        // The point of appending is a recording a player can still use. Each
        // session restarts its own timestamps and the muxer marks the join with
        // an EXT-X-DISCONTINUITY, so this is where that either works or does
        // not.
        // What a player actually builds its timeline from: the durations and
        // the wall-clock anchors, one per session.
        {
            const std::string pl =
                    read_file(std::filesystem::path(rstorage.playlist()));
            double total = 0;
            for (size_t at = pl.find("#EXTINF:"); at != std::string::npos;
                 at = pl.find("#EXTINF:", at + 1)) {
                total += atof(pl.c_str() + at + strlen("#EXTINF:"));
            }
            const size_t dates = count_occurrences(pl, "#EXT-X-PROGRAM-DATE-TIME:");
            const size_t joins = count_occurrences(pl, "#EXT-X-DISCONTINUITY");
            const size_t maps = count_occurrences(pl, "#EXT-X-MAP:");
            fprintf(stderr,
                    "appended playlist: %.3f seconds, %zu date-times, %zu "
                    "discontinuities, %zu maps\n",
                    total, dates, joins, maps);

            // Three sessions of the same capture: the durations have to add up,
            // because #EXTINF is the timeline a player seeks in.
            CHECK(total > 2.99 * total_seconds,
                  "the appended playlist covers %.3f seconds, want about three "
                  "times %.3f",
                  total, total_seconds);
            // One wall-clock anchor per session. This is what a viewer seeks by
            // when an event names a moment.
            CHECK(dates == 3, "%zu date-times for three sessions", dates);
            // One join per reconnect, and none before the first session: an
            // unmarked timestamp reset is what breaks a player.
            CHECK(joins == 2, "%zu discontinuities for two reconnects", joins);
            // One initialisation segment, shared. More than one would mean the
            // fragments disagree about their sample description, which
            // StreamStorage::reconcile exists to prevent.
            CHECK(maps == 1, "%zu EXT-X-MAP entries, want one", maps);
        }

        Playback joined;
        const int jrc = verify_playback(rstorage.playlist(), &joined);
        CHECK(jrc == 0, "reading back the appended playlist: %s",
              av_error(jrc).c_str());
        if (jrc == 0) {
            fprintf(stderr,
                    "appended playback: %s %dx%d, %lld packets, %lld keyframes,"
                    " %.3f seconds, increasing=%d\n",
                    avcodec_get_name(joined.codec), joined.width, joined.height,
                    static_cast<long long>(joined.packets),
                    static_cast<long long>(joined.keyframes), joined.seconds,
                    joined.timestamps_increase ? 1 : 0);
            CHECK(joined.codec == AV_CODEC_ID_H264, "the appended output is %s",
                  avcodec_get_name(joined.codec));
            CHECK(joined.width > 0 && joined.height > 0,
                  "the appended output reports %dx%d", joined.width,
                  joined.height);
            // Three sessions of the same capture, so three times the access
            // units: nothing may be lost across a join.
            CHECK(joined.packets == static_cast<int64_t>(3 * stats.written),
                  "read back %lld access units from three sessions of %llu",
                  static_cast<long long>(joined.packets),
                  static_cast<unsigned long long>(stats.written));

            // Deliberately not asserting joined.timestamps_increase or
            // joined.seconds here. Each session restarts its own presentation
            // clock and the playlist marks the reset with an
            // EXT-X-DISCONTINUITY, which is what the format requires; hls.js
            // and Safari re-base on it, and libavformat's HLS demuxer does not,
            // so this reader sees overlapping clocks and one session's
            // duration. The playlist checks above are what a player goes by.
            CHECK(joined.keyframes >= 3,
                  "the appended output holds %lld keyframes, so a join is not "
                  "seekable",
                  static_cast<long long>(joined.keyframes));
        }
    }

    // Appending and retention have to hold at once. The muxer cannot see the
    // fragments that fell off a playlist it had not read, so without a reap it
    // orphans one per reconnect and the directory grows without bound while the
    // playlist stays honest.
    {
        StreamStorage bstorage((root / "bounded").string(), "replay", "camera");
        // One-second fragments, two-second retention: the capture outruns the
        // window on every pass.
        const HlsOptions bopts{1, 2, false};

        for (int attempt = 1; attempt <= 4; attempt++) {
            ReplayFrameSource frames_again(capture.packets);
            StreamStats again;
            const int arc = run_stream(capture.sdp, frames_again, bstorage, bopts,
                                       &again);
            CHECK(arc == 0, "run_stream (bounded %d): %s", attempt,
                  av_error(arc).c_str());
            if (arc != 0) {
                break;
            }

            const size_t listed = count_occurrences(
                    read_file(std::filesystem::path(bstorage.playlist())),
                    "#EXTINF:");
            size_t on_disk_now = 0;
            for (const auto &e :
                 std::filesystem::directory_iterator(bstorage.directory())) {
                if (e.path().extension() == ".m4s") {
                    on_disk_now++;
                }
            }
            fprintf(stderr, "bounded %d: listed=%zu on_disk=%zu\n", attempt,
                    listed, on_disk_now);

            CHECK(listed > 0, "attempt %d lists nothing", attempt);
            // hls_delete_threshold keeps one unlisted fragment for a player
            // still fetching it; anything beyond that is an orphan.
            CHECK(on_disk_now <= listed + 1,
                  "attempt %d left %zu fragments on disk for a playlist of "
                  "%zu, so reconnects are orphaning them",
                  attempt, on_disk_now, listed);
        }
    }

    if (failures == 0) {
        // CAMS_TEST_KEEP leaves the output in place on success too, which is
        // what makes a playlist diffable between two runs.
        if (getenv("CAMS_TEST_KEEP") == nullptr) {
            std::filesystem::remove_all(root);
        } else {
            fprintf(stderr, "output kept under %s\n", root.string().c_str());
        }
        fprintf(stderr, "PASS\n");
        return 0;
    }
    fprintf(stderr, "%d check(s) failed; output left under %s\n",
            failures, root.string().c_str());
    return 1;
}
