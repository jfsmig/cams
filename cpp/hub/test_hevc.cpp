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
// The H.265 arm of the remux chain, against a stream made here.
//
// There is no recorded HEVC capture the way cams-capture-879216526.tar covers
// H.264, and no H.265 camera to make one from. So this builds the stream
// instead: libx265 encodes a few synthetic frames, libavformat's RTP muxer
// packetises them per RFC 7798 and reports the session description it would
// have advertised, and the result is fed through the same run_stream a camera
// reaches.
//
// That is not a substitute for a real camera -- a device's parameter sets,
// fragmentation and timing are its own -- but it does settle the four things
// the arm actually depends on: that the description survives the reduction,
// that avformat_find_stream_info fills the picture size for HEVC as it does for
// H.264, that the stages above the sink are codec-agnostic, and that the mov
// muxer writes an hvcC the output can be read back through.
//
// The encoder is a test dependency and not a pipeline one. If the image has no
// libx265 the test reports that and passes, rather than failing for a reason
// that says nothing about this code.
//

#include <cstdarg>
#include <cstdio>
#include <cstdlib>
#include <cstring>
#include <filesystem>
#include <string>
#include <vector>
#include <unistd.h>

extern "C" {
#include <libavcodec/avcodec.h>
#include <libavformat/avformat.h>
#include <libavutil/imgutils.h>
#include <libavutil/opt.h>
}

#include "AvError.hpp"
#include "FrameSource.hpp"
#include "MediaPipeline.hpp"
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

constexpr int kWidth = 320;
constexpr int kHeight = 240;
constexpr int kFrames = 20;
constexpr int kFrameRate = 25;

// One RTP datagram, as the muxer handed it over.
struct Datagram {
    std::vector<uint8_t> body;
};

struct Capture {
    std::string sdp;
    std::vector<Datagram> packets;
};

// ------------------------------------------------------------------ encoding

// A moving gradient: something for the encoder to chew on that is not a flat
// field, so the parameter sets and the frame sizes are realistic enough.
void paint(AVFrame *frame, int at) {
    for (int y = 0; y < kHeight; y++) {
        for (int x = 0; x < kWidth; x++) {
            frame->data[0][y * frame->linesize[0] + x] =
                    static_cast<uint8_t>(x + y + at * 3);
        }
    }
    for (int y = 0; y < kHeight / 2; y++) {
        for (int x = 0; x < kWidth / 2; x++) {
            frame->data[1][y * frame->linesize[1] + x] =
                    static_cast<uint8_t>(128 + at);
            frame->data[2][y * frame->linesize[2] + x] =
                    static_cast<uint8_t>(64 + x);
        }
    }
}

// encode fills out with HEVC access units. Returns 0, a libav error, or
// AVERROR_ENCODER_NOT_FOUND when the image has no libx265.
int encode(std::vector<AVPacket *> *out, AVCodecParameters **par,
           AVRational *time_base) {
    const AVCodec *codec = avcodec_find_encoder_by_name("libx265");
    if (codec == nullptr) {
        return AVERROR_ENCODER_NOT_FOUND;
    }

    AVCodecContext *ctx = avcodec_alloc_context3(codec);
    if (ctx == nullptr) {
        return AVERROR(ENOMEM);
    }
    ctx->width = kWidth;
    ctx->height = kHeight;
    ctx->pix_fmt = AV_PIX_FMT_YUV420P;
    ctx->time_base = AVRational{1, kFrameRate};
    ctx->framerate = AVRational{kFrameRate, 1};
    // Short GOP so the replay holds several keyframes, and quiet and quick
    // because none of that is what is being tested. Headers stay in band,
    // which is what a camera does and what the RTP packetiser needs.
    ctx->gop_size = 5;
    av_opt_set(ctx->priv_data, "preset", "ultrafast", 0);
    av_opt_set(ctx->priv_data, "x265-params",
               "keyint=5:min-keyint=5:scenecut=0:log-level=none:repeat-headers=1",
               0);

    int rc = avcodec_open2(ctx, codec, nullptr);
    if (rc < 0) {
        avcodec_free_context(&ctx);
        return rc;
    }

    AVFrame *frame = av_frame_alloc();
    if (frame == nullptr) {
        avcodec_free_context(&ctx);
        return AVERROR(ENOMEM);
    }
    frame->format = ctx->pix_fmt;
    frame->width = ctx->width;
    frame->height = ctx->height;
    rc = av_frame_get_buffer(frame, 0);
    if (rc < 0) {
        av_frame_free(&frame);
        avcodec_free_context(&ctx);
        return rc;
    }

    for (int at = 0; at <= kFrames; at++) {
        AVFrame *sending = nullptr;
        if (at < kFrames) {
            rc = av_frame_make_writable(frame);
            if (rc < 0) {
                break;
            }
            paint(frame, at);
            frame->pts = at;
            sending = frame;
        }

        rc = avcodec_send_frame(ctx, sending); // null flushes
        if (rc < 0) {
            break;
        }
        for (;;) {
            AVPacket *pkt = av_packet_alloc();
            if (pkt == nullptr) {
                rc = AVERROR(ENOMEM);
                break;
            }
            const int prc = avcodec_receive_packet(ctx, pkt);
            if (prc == AVERROR(EAGAIN) || prc == AVERROR_EOF) {
                av_packet_free(&pkt);
                break;
            }
            if (prc < 0) {
                av_packet_free(&pkt);
                rc = prc;
                break;
            }
            out->push_back(pkt);
        }
        if (rc < 0) {
            break;
        }
    }

    if (rc >= 0) {
        *par = avcodec_parameters_alloc();
        if (*par == nullptr) {
            rc = AVERROR(ENOMEM);
        } else {
            rc = avcodec_parameters_from_context(*par, ctx);
            *time_base = ctx->time_base;
        }
    }

    av_frame_free(&frame);
    avcodec_free_context(&ctx);
    return rc < 0 ? rc : 0;
}

// ----------------------------------------------------------------- packetising

// The RTP muxer writes one datagram per flush, so a write callback is where the
// packet boundaries are.
int on_datagram(void *opaque, const uint8_t *buf, int len) {
    auto *into = static_cast<std::vector<Datagram> *>(opaque);
    if (len > 0) {
        into->push_back(Datagram{std::vector<uint8_t>(buf, buf + len)});
    }
    return len;
}

int packetise(const std::vector<AVPacket *> &units, const AVCodecParameters *par,
              AVRational time_base, Capture *out) {
    AVFormatContext *ctx = nullptr;
    int rc = avformat_alloc_output_context2(&ctx, nullptr, "rtp", nullptr);
    if (rc < 0) {
        return rc;
    }

    AVStream *st = avformat_new_stream(ctx, nullptr);
    if (st == nullptr) {
        avformat_free_context(ctx);
        return AVERROR(ENOMEM);
    }
    rc = avcodec_parameters_copy(st->codecpar, par);
    if (rc < 0) {
        avformat_free_context(ctx);
        return rc;
    }
    st->time_base = time_base;

    constexpr int kBufferSize = 65536;
    auto *buffer = static_cast<uint8_t *>(av_malloc(kBufferSize));
    if (buffer == nullptr) {
        avformat_free_context(ctx);
        return AVERROR(ENOMEM);
    }
    AVIOContext *io = avio_alloc_context(buffer, kBufferSize, 1, &out->packets,
                                         nullptr, &on_datagram, nullptr);
    if (io == nullptr) {
        av_free(buffer);
        avformat_free_context(ctx);
        return AVERROR(ENOMEM);
    }
    // The RTP muxer takes its payload size from the context, and a custom one
    // reports zero -- which it refuses as "Max packet size 0 too low". This is
    // also what makes it fragment, which is the interesting half: a camera's
    // access units arrive in pieces.
    io->max_packet_size = 1400;
    // Presetting pb is also what sets AVFMT_FLAG_CUSTOM_IO, so the context
    // stays ours to free.
    ctx->pb = io;

    rc = avformat_write_header(ctx, nullptr);
    if (rc < 0) {
        av_freep(&io->buffer);
        avio_context_free(&io);
        avformat_free_context(ctx);
        return rc;
    }

    // The description the muxer would have advertised, which is the same shape
    // a camera publishes: no address, no port.
    char sdp[4096] = {0};
    const int src = av_sdp_create(&ctx, 1, sdp, sizeof(sdp));
    if (src >= 0) {
        out->sdp = sdp;
    }

    for (AVPacket *pkt : units) {
        AVPacket *copy = av_packet_clone(pkt);
        if (copy == nullptr) {
            rc = AVERROR(ENOMEM);
            break;
        }
        copy->stream_index = 0;
        const int wrc = av_write_frame(ctx, copy);
        av_packet_free(&copy);
        if (wrc < 0) {
            rc = wrc;
            break;
        }
    }
    if (rc >= 0) {
        rc = av_write_trailer(ctx);
    }

    av_freep(&io->buffer);
    avio_context_free(&io);
    avformat_free_context(ctx);
    if (rc >= 0 && src < 0) {
        return src;
    }
    return rc;
}

// -------------------------------------------------------------------- replay

class ReplaySource : public FrameSource {
public:
    explicit ReplaySource(const std::vector<Datagram> &packets)
            : packets_{packets} {}

    int next(uint8_t *buf, int cap) override {
        while (at_ < packets_.size()) {
            const Datagram &p = packets_[at_++];
            if (p.body.empty() || p.body.size() > static_cast<size_t>(cap)) {
                continue;
            }
            memcpy(buf, p.body.data(), p.body.size());
            delivered_++;
            return static_cast<int>(p.body.size());
        }
        return AVERROR_EOF;
    }

    [[nodiscard]] size_t delivered() const { return delivered_; }

private:
    const std::vector<Datagram> &packets_;
    size_t at_{0};
    size_t delivered_{0};
};

// What reading the output back reports.
struct Playback {
    AVCodecID codec{AV_CODEC_ID_NONE};
    int width{0};
    int height{0};
    int64_t packets{0};
    int64_t keyframes{0};
};

int read_back(const std::string &playlist, Playback *out) {
    AVFormatContext *ctx = nullptr;
    int rc = avformat_open_input(&ctx, playlist.c_str(), nullptr, nullptr);
    if (rc < 0) {
        return rc;
    }
    rc = avformat_find_stream_info(ctx, nullptr);
    if (rc < 0) {
        avformat_close_input(&ctx);
        return rc;
    }
    if (ctx->nb_streams != 1) {
        avformat_close_input(&ctx);
        return AVERROR_INVALIDDATA;
    }

    const AVCodecParameters *par = ctx->streams[0]->codecpar;
    out->codec = par->codec_id;
    out->width = par->width;
    out->height = par->height;

    AVPacket *pkt = av_packet_alloc();
    if (pkt == nullptr) {
        avformat_close_input(&ctx);
        return AVERROR(ENOMEM);
    }
    while (av_read_frame(ctx, pkt) >= 0) {
        out->packets++;
        if ((pkt->flags & AV_PKT_FLAG_KEY) != 0) {
            out->keyframes++;
        }
        av_packet_unref(pkt);
    }
    av_packet_free(&pkt);
    avformat_close_input(&ctx);
    return 0;
}

} // namespace

int main() {
    av_log_set_level(AV_LOG_ERROR);
    // CAMS_TEST_VERBOSE turns libav's own account of a failure back on, which
    // is most of the diagnosis when a muxer refuses a packet.
    if (getenv("CAMS_TEST_VERBOSE") != nullptr) {
        av_log_set_level(AV_LOG_DEBUG);
    }

    std::vector<AVPacket *> units;
    AVCodecParameters *par = nullptr;
    AVRational time_base{1, kFrameRate};

    const int erc = encode(&units, &par, &time_base);
    if (erc == AVERROR_ENCODER_NOT_FOUND) {
        fprintf(stderr,
                "SKIP: this build of libavcodec has no libx265, so no H.265 "
                "stream can be made here\nPASS\n");
        return 0;
    }
    if (erc < 0) {
        fprintf(stderr, "encoding: %s\n", av_error(erc).c_str());
        return 1;
    }
    fprintf(stderr, "encoded %zu access units of %dx%d HEVC\n", units.size(),
            kWidth, kHeight);

    Capture capture;
    const int prc = packetise(units, par, time_base, &capture);
    for (AVPacket *pkt : units) {
        AVPacket *p = pkt;
        av_packet_free(&p);
    }
    avcodec_parameters_free(&par);
    if (prc < 0) {
        fprintf(stderr, "packetising: %s\n", av_error(prc).c_str());
        return 1;
    }
    fprintf(stderr, "packetised into %zu RTP datagrams\n", capture.packets.size());

    CHECK(!capture.sdp.empty(), "the muxer produced no session description");
    CHECK(!capture.packets.empty(), "the muxer produced no RTP packet");
    // The description has to name H.265, or the rest proves nothing.
    CHECK(capture.sdp.find("H265") != std::string::npos,
          "the description does not mention H265:\n%s", capture.sdp.c_str());
    if (failures != 0) {
        return 1;
    }

    const auto root = std::filesystem::temp_directory_path() /
                      ("cams-hevc-" + std::to_string(::getpid()));
    std::filesystem::remove_all(root);

    StreamStorage storage(root.string(), "replay", "camera");
    const HlsOptions opts{1, 3600, true};

    ReplaySource frames(capture.packets);
    StreamStats stats;

    const int rc = run_stream(capture.sdp, frames, storage, opts, &stats);
    CHECK(rc == 0, "run_stream: %s", av_error(rc).c_str());
    fprintf(stderr,
            "delivered=%zu written=%llu skipped=%llu undated=%llu track=%u\n",
            frames.delivered(),
            static_cast<unsigned long long>(stats.written),
            static_cast<unsigned long long>(stats.skipped),
            static_cast<unsigned long long>(stats.undated), stats.track);

    if (rc == 0) {
        CHECK(stats.written > 0, "no access unit reached a fragment");
        // The parameter-set-only unit is the one that arrives undated. More
        // than a handful would mean the depacketiser is producing something
        // this pipeline does not understand.
        CHECK(stats.undated <= 2, "%llu access units arrived undated",
              static_cast<unsigned long long>(stats.undated));

        // The initialisation segment is where hvcC lands, and an fMP4 track
        // cannot be declared without the picture size -- which the description
        // does not carry, so RtpSource had to settle it for HEVC too.
        CHECK(std::filesystem::exists(storage.init()),
              "no initialisation segment at %s", storage.init().c_str());

        Playback back;
        const int brc = read_back(storage.playlist(), &back);
        CHECK(brc == 0, "reading the output back: %s", av_error(brc).c_str());
        if (brc == 0) {
            fprintf(stderr,
                    "playback: %s %dx%d, %lld packets, %lld keyframes\n",
                    avcodec_get_name(back.codec), back.width, back.height,
                    static_cast<long long>(back.packets),
                    static_cast<long long>(back.keyframes));

            CHECK(back.codec == AV_CODEC_ID_HEVC,
                  "the output codec is %s, want hevc",
                  avcodec_get_name(back.codec));
            CHECK(back.width == kWidth && back.height == kHeight,
                  "the output reports %dx%d, want %dx%d", back.width,
                  back.height, kWidth, kHeight);
            CHECK(back.packets > 0, "the output yielded no packet");
            CHECK(back.keyframes > 0,
                  "the output holds no keyframe, so it cannot be joined");
        }
    }

    if (failures == 0) {
        std::filesystem::remove_all(root);
        fprintf(stderr, "PASS\n");
        return 0;
    }
    fprintf(stderr, "%d check(s) failed; output left under %s\n", failures,
            root.string().c_str());
    return 1;
}
