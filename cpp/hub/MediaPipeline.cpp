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

#include "MediaPipeline.hpp"

#include <cstdint>
#include <cstdio>
#include <string>

#include "AvError.hpp"
#include "DurationFiller.hpp"
#include "KeyframeGate.hpp"
#include "RtpSource.hpp"
#include "Sdp.hpp"

namespace {

// What the directory is recording, in one line: enough that two sessions can
// tell whether their fragments belong to the same playlist.
//
// The parameter sets are folded in rather than just the dimensions, because a
// changed SPS -- another level, another profile -- yields an initialisation
// segment the older fragments no longer match, at the same width and height.
std::string recording_of(const AVCodecParameters *par) {
    uint64_t hash = 1469598103934665603ULL; // FNV-1a
    for (int i = 0; i < par->extradata_size; i++) {
        hash ^= par->extradata[i];
        hash *= 1099511628211ULL;
    }

    char line[192];
    snprintf(line, sizeof(line), "%s %dx%d %d:%016llx",
             avcodec_get_name(par->codec_id), par->width, par->height,
             par->extradata_size, static_cast<unsigned long long>(hash));
    return line;
}

// Which codecs the remux chain can carry into fMP4.
//
// Both go through untouched. libavformat's mov muxer writes avcC or hvcC from
// the Annex-B parameter sets itself, and every stage above the sink is
// codec-agnostic: a keyframe is a keyframe and a timestamp delta is a timestamp
// delta.
//
// What differs is downstream of the hub rather than inside it. H.265 in fMP4
// plays natively in Safari and in a Chrome with hardware decode, and in little
// else, so the agent prefers a camera's H.264 profile whenever it offers one --
// see chooseProfile. This arm carries the cameras that offer nothing else,
// which is why it is worth having and why it is not the common path.
bool codec_is_remuxable(AVCodecID id) {
    return id == AV_CODEC_ID_H264 || id == AV_CODEC_ID_HEVC;
}

} // namespace

bool track_of(std::string_view sdp, uint32_t *track) {
    const VideoMedia media = keep_first_video_media(sdp);
    if (media.sdp.empty()) {
        return false;
    }
    if (track != nullptr) {
        *track = media.track;
    }
    return true;
}

int run_stream(std::string_view sdp, FrameSource &frames,
               StreamStorage &storage, const HlsOptions &opts,
               StreamStats *stats) {
    const VideoMedia media = keep_first_video_media(sdp);
    if (media.sdp.empty()) {
        fprintf(stderr, "cams: the session description carries no video\n");
        return AVERROR(EINVAL);
    }
    if (stats != nullptr) {
        stats->track = media.track;
    }

    int rc = storage.prepare();
    if (rc < 0) {
        fprintf(stderr, "cams: storage: %s\n", av_error(rc).c_str());
        return rc;
    }

    RtpSource source(media.sdp, frames);
    rc = source.open();
    if (rc < 0) {
        fprintf(stderr, "cams: rtp source: %s\n", av_error(rc).c_str());
        return rc;
    }
    // The description was reduced to one media, so anything else means the
    // demuxer read it differently than keep_first_video_media did.
    if (source.context()->nb_streams != 1) {
        fprintf(stderr, "cams: the description yielded %u streams, want 1\n",
                source.context()->nb_streams);
        return AVERROR(EINVAL);
    }

    // Which chain to build is decided here, once, from the description, and
    // never revisited: with fMP4 the initialisation segment pins the codec
    // parameters, so a codec change mid-stream would need a new one plus an
    // EXT-X-DISCONTINUITY, which players disagree about. A camera that changes
    // codec gets a new stream instead.
    //
    // Only the remux chain exists, and it carries both H.264 and H.265. A
    // transcode chain -- MJPEG in, or H.265 out to a browser that cannot play
    // it -- would *replace* KeyframeGate and DurationFiller rather than join
    // them, because an encoder starts on a keyframe by construction and dates
    // its own output. Until it is built, a codec that needs one is refused
    // here by name rather than left to produce a file nothing can play.
    const AVStream *in = source.context()->streams[0];
    if (!codec_is_remuxable(in->codecpar->codec_id)) {
        fprintf(stderr,
                "cams: the stream carries %s, which needs a transcode stage "
                "that is not built\n",
                avcodec_get_name(in->codecpar->codec_id));
        return AVERROR_INVALIDDATA;
    }

    // Whether this session extends the recording already in the directory or
    // replaces it. Settled before anything is written, because it decides
    // whether the playlist is appended to or truncated.
    StreamStorage::Continuity how = StreamStorage::Continuity::Fresh;
    rc = storage.reconcile(recording_of(in->codecpar), &how);
    if (rc < 0) {
        fprintf(stderr, "cams: storage: %s\n", av_error(rc).c_str());
        return rc;
    }
    if (stats != nullptr) {
        stats->appended = how == StreamStorage::Continuity::Append;
    }

    // The remux chain, from the sink up. Declaring the sink first is what
    // makes it outlive the stages holding a reference to it, since destruction
    // runs in reverse.
    FragmentSink sink(storage, opts, how);
    DurationFiller filler(sink);
    KeyframeGate gate(filler);
    PacketStage &head = gate;

    rc = head.open(in->codecpar, in->time_base);
    if (rc < 0) {
        fprintf(stderr, "cams: pipeline: %s\n", av_error(rc).c_str());
        return rc;
    }

    AVPacket *pkt = av_packet_alloc();
    if (pkt == nullptr) {
        return AVERROR(ENOMEM);
    }

    int result = 0;
    for (;;) {
        rc = source.read(pkt);
        if (rc == AVERROR_EOF) {
            break;
        }
        if (rc < 0) {
            fprintf(stderr, "cams: read: %s\n", av_error(rc).c_str());
            result = rc;
            break;
        }
        // write consumes the packet either way, so there is nothing to unref.
        rc = head.write(pkt);
        if (rc < 0) {
            fprintf(stderr, "cams: write: %s\n", av_error(rc).c_str());
            result = rc;
            break;
        }
    }
    av_packet_free(&pkt);

    // The trailer is what completes the playlist, so it is written even for a
    // stream that ended badly: the segments already on disk are playable and a
    // playlist without an end marker is not.
    rc = head.finish();
    if (rc < 0) {
        fprintf(stderr, "cams: finish: %s\n", av_error(rc).c_str());
        if (result == 0) {
            result = rc;
        }
    }

    if (stats != nullptr) {
        stats->written = sink.written();
        stats->skipped = gate.skipped();
        stats->undated = filler.undated();
        stats->rtcp_discarded = source.rtcp_discarded();
    }
    return result;
}
