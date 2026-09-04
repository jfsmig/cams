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
// The pipeline stages, on their own.
//
// run_stream is covered end to end by test_rtp2hls, which needs a capture and
// libav to say anything at all. These two stages are pure policy over
// AVPackets, so they can be driven directly -- which is most of the reason for
// having pulled them out of the sink.
//

#include <cstdarg>
#include <cstdio>
#include <vector>

extern "C" {
#include <libavcodec/packet.h>
#include <libavformat/avformat.h>
}

#include "DurationFiller.hpp"
#include "KeyframeGate.hpp"

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

// Counts instead of aborting, so one run reports everything that is wrong.
#define CHECK(cond, ...)                                                       \
    do {                                                                       \
        if (!(cond)) {                                                         \
            fail(__FILE__, __LINE__, #cond, __VA_ARGS__);                      \
        }                                                                      \
    } while (0)

// What reached the end of the chain.
struct Seen {
    int64_t pts;
    int64_t duration;
    int flags;
};

// Recorder is a terminal stage that remembers what it was given. It honours the
// PacketStage contract and releases every packet.
class Recorder : public PacketStage {
public:
    int open(const AVCodecParameters *in, AVRational in_tb) override {
        opens_++;
        in_ = in;
        tb_ = in_tb;
        return 0;
    }

    int write(AVPacket *pkt) override {
        seen_.push_back({pkt->pts, pkt->duration, pkt->flags});
        av_packet_unref(pkt);
        return fail_write_ ? AVERROR(EIO) : 0;
    }

    int finish() override {
        finishes_++;
        return 0;
    }

    void fail_writes() { fail_write_ = true; }

    std::vector<Seen> seen_;
    int opens_{0};
    int finishes_{0};
    const AVCodecParameters *in_{nullptr};
    AVRational tb_{0, 1};

private:
    bool fail_write_{false};
};

// A packet carrying a real buffer, so that av_packet_move_ref has something to
// move.
AVPacket *packet(int64_t pts, bool key) {
    AVPacket *pkt = av_packet_alloc();
    if (pkt == nullptr || av_new_packet(pkt, 4) < 0) {
        fprintf(stderr, "out of memory building a packet\n");
        exit(1);
    }
    pkt->pts = pts;
    pkt->dts = pts;
    pkt->duration = 0;
    pkt->flags = key ? AV_PKT_FLAG_KEY : 0;
    return pkt;
}

void free_packet(AVPacket **pkt) { av_packet_free(pkt); }

// ---------------------------------------------------------------- KeyframeGate

void gate_drops_until_the_first_keyframe() {
    Recorder rec;
    KeyframeGate gate(rec);

    CHECK(gate.open(nullptr, AVRational{1, 90000}) == 0, "open failed");
    CHECK(rec.opens_ == 1, "the gate did not open its downstream (%d)", rec.opens_);

    // Two delta frames, then a keyframe, then a delta frame.
    const bool keys[] = {false, false, true, false};
    for (int i = 0; i < 4; i++) {
        AVPacket *p = packet(i * 100, keys[i]);
        CHECK(gate.write(p) == 0, "write %d failed", i);
        free_packet(&p);
    }

    CHECK(gate.skipped() == 2, "skipped %llu, want 2",
          static_cast<unsigned long long>(gate.skipped()));
    CHECK(rec.seen_.size() == 2, "forwarded %zu packets, want 2", rec.seen_.size());
    if (rec.seen_.size() == 2) {
        CHECK(rec.seen_[0].pts == 200, "the first forwarded packet is at %lld, "
                                       "want the keyframe at 200",
              static_cast<long long>(rec.seen_[0].pts));
        // Everything after the first keyframe passes, keyframe or not.
        CHECK(rec.seen_[1].pts == 300, "the second forwarded packet is at %lld",
              static_cast<long long>(rec.seen_[1].pts));
    }

    CHECK(gate.finish() == 0, "finish failed");
    CHECK(rec.finishes_ == 1, "the gate did not finish its downstream");
}

void gate_passes_everything_once_open() {
    Recorder rec;
    KeyframeGate gate(rec);
    CHECK(gate.open(nullptr, AVRational{1, 90000}) == 0, "open failed");

    AVPacket *first = packet(0, true);
    CHECK(gate.write(first) == 0, "write failed");
    free_packet(&first);

    for (int i = 1; i < 5; i++) {
        AVPacket *p = packet(i * 100, false);
        CHECK(gate.write(p) == 0, "write %d failed", i);
        free_packet(&p);
    }
    CHECK(gate.skipped() == 0, "skipped %llu after a keyframe, want 0",
          static_cast<unsigned long long>(gate.skipped()));
    CHECK(rec.seen_.size() == 5, "forwarded %zu, want 5", rec.seen_.size());
}

void gate_reports_a_downstream_failure() {
    Recorder rec;
    rec.fail_writes();
    KeyframeGate gate(rec);
    CHECK(gate.open(nullptr, AVRational{1, 90000}) == 0, "open failed");

    AVPacket *p = packet(0, true);
    CHECK(gate.write(p) < 0, "a failing downstream write was reported as success");
    free_packet(&p);
}

// -------------------------------------------------------------- DurationFiller

void filler_dates_each_packet_from_its_successor() {
    Recorder rec;
    DurationFiller filler(rec);
    CHECK(filler.open(nullptr, AVRational{1, 90000}) == 0, "open failed");
    CHECK(rec.opens_ == 1, "the filler did not open its downstream");

    // Three packets, 3000 ticks apart.
    for (int i = 0; i < 3; i++) {
        AVPacket *p = packet(i * 3000, i == 0);
        CHECK(filler.write(p) == 0, "write %d failed", i);
        free_packet(&p);
    }

    // One is always held back, so two of the three are out by now.
    CHECK(rec.seen_.size() == 2, "emitted %zu of 3 before finish, want 2",
          rec.seen_.size());
    if (rec.seen_.size() == 2) {
        CHECK(rec.seen_[0].duration == 3000, "packet 0 dated %lld, want 3000",
              static_cast<long long>(rec.seen_[0].duration));
        CHECK(rec.seen_[1].duration == 3000, "packet 1 dated %lld, want 3000",
              static_cast<long long>(rec.seen_[1].duration));
    }

    CHECK(filler.finish() == 0, "finish failed");
    CHECK(rec.seen_.size() == 3, "the held packet was not flushed (%zu)",
          rec.seen_.size());
    if (rec.seen_.size() == 3) {
        // Nothing follows the last one, so it inherits the interval before it.
        CHECK(rec.seen_[2].duration == 3000,
              "the last packet went out with duration %lld, want the inherited "
              "3000",
              static_cast<long long>(rec.seen_[2].duration));
    }
    CHECK(rec.finishes_ == 1, "the filler did not finish its downstream");
}

void filler_overwrites_the_one_tick_libav_gives_the_last_packet() {
    Recorder rec;
    DurationFiller filler(rec);
    CHECK(filler.open(nullptr, AVRational{1, 90000}) == 0, "open failed");

    AVPacket *a = packet(0, true);
    CHECK(filler.write(a) == 0, "write a failed");
    free_packet(&a);

    AVPacket *b = packet(3000, false);
    CHECK(filler.write(b) == 0, "write b failed");
    free_packet(&b);

    // This is what libavformat hands the final access unit: one tick, eleven
    // microseconds at 90 kHz, which is not a frame.
    AVPacket *last = packet(6000, false);
    last->duration = 1;
    CHECK(filler.write(last) == 0, "write last failed");
    free_packet(&last);

    CHECK(filler.finish() == 0, "finish failed");
    CHECK(rec.seen_.size() == 3, "emitted %zu, want 3", rec.seen_.size());
    if (rec.seen_.size() == 3) {
        CHECK(rec.seen_[2].duration == 3000,
              "the last packet kept duration %lld; the bogus one tick was not "
              "replaced",
              static_cast<long long>(rec.seen_[2].duration));
    }
}

void filler_on_a_single_packet_emits_it() {
    Recorder rec;
    DurationFiller filler(rec);
    CHECK(filler.open(nullptr, AVRational{1, 90000}) == 0, "open failed");

    AVPacket *only = packet(0, true);
    CHECK(filler.write(only) == 0, "write failed");
    free_packet(&only);
    CHECK(rec.seen_.empty(), "the only packet went out before finish");

    CHECK(filler.finish() == 0, "finish failed");
    CHECK(rec.seen_.size() == 1, "emitted %zu, want 1", rec.seen_.size());
    if (rec.seen_.size() == 1) {
        // There was never an interval to inherit, so it stays as it arrived.
        CHECK(rec.seen_[0].duration == 0, "a lone packet was dated %lld",
              static_cast<long long>(rec.seen_[0].duration));
    }
}

// The opening access unit always arrives undated -- the depacketiser has no
// clock reference for it yet -- and it is a keyframe. Dropping it would leave
// the first fragment starting on a delta frame.
void filler_dates_the_opening_access_unit() {
    Recorder rec;
    DurationFiller filler(rec);
    CHECK(filler.open(nullptr, AVRational{1, 90000}) == 0, "open failed");

    AVPacket *first = packet(0, true);
    first->pts = AV_NOPTS_VALUE;
    first->dts = AV_NOPTS_VALUE;
    CHECK(filler.write(first) == 0, "the opening packet was refused");
    free_packet(&first);

    for (int i = 1; i < 3; i++) {
        AVPacket *p = packet(i * 3000, false);
        CHECK(filler.write(p) == 0, "write %d failed", i);
        free_packet(&p);
    }
    CHECK(filler.finish() == 0, "finish failed");

    CHECK(filler.undated() == 0, "the opening packet was counted as dropped");
    CHECK(rec.seen_.size() == 3, "emitted %zu, want 3", rec.seen_.size());
    if (rec.seen_.size() == 3) {
        // Dated to zero, which is where the depacketiser's unwrapped clock
        // starts, and still the keyframe the fragment has to begin on.
        CHECK(rec.seen_[0].pts == 0, "the opening packet went out at %lld",
              static_cast<long long>(rec.seen_[0].pts));
        CHECK((rec.seen_[0].flags & AV_PKT_FLAG_KEY) != 0,
              "the opening keyframe was not the first packet out");
        // And its duration follows from its successor like any other.
        CHECK(rec.seen_[0].duration == 3000, "the opening packet lasts %lld",
              static_cast<long long>(rec.seen_[0].duration));
    }
}

// Mid-stream there is nothing to derive a timestamp from, and the muxer refuses
// the packet outright rather than warning.
void filler_drops_an_undated_packet_mid_stream() {
    Recorder rec;
    DurationFiller filler(rec);
    CHECK(filler.open(nullptr, AVRational{1, 90000}) == 0, "open failed");

    AVPacket *first = packet(0, true);
    CHECK(filler.write(first) == 0, "write failed");
    free_packet(&first);

    AVPacket *undated = packet(3000, false);
    undated->pts = AV_NOPTS_VALUE;
    undated->dts = AV_NOPTS_VALUE;
    CHECK(filler.write(undated) == 0, "an undated packet was reported as an error");
    free_packet(&undated);
    CHECK(filler.undated() == 1, "counted %llu undated, want 1",
          static_cast<unsigned long long>(filler.undated()));

    // The stream carries on around it.
    for (int i = 2; i < 4; i++) {
        AVPacket *p = packet(i * 3000, false);
        CHECK(filler.write(p) == 0, "write %d failed", i);
        free_packet(&p);
    }
    CHECK(filler.finish() == 0, "finish failed");

    CHECK(rec.seen_.size() == 3, "emitted %zu, want the 3 dated ones",
          rec.seen_.size());
    for (const auto &seen : rec.seen_) {
        CHECK(seen.pts != AV_NOPTS_VALUE, "an undated packet reached the sink");
    }
}

void filler_finish_is_idempotent() {
    Recorder rec;
    DurationFiller filler(rec);
    CHECK(filler.open(nullptr, AVRational{1, 90000}) == 0, "open failed");

    AVPacket *p = packet(0, true);
    CHECK(filler.write(p) == 0, "write failed");
    free_packet(&p);

    CHECK(filler.finish() == 0, "first finish failed");
    CHECK(filler.finish() == 0, "second finish failed");
    CHECK(rec.seen_.size() == 1, "the held packet was emitted %zu times",
          rec.seen_.size());
}

// ---------------------------------------------------------------- the two together

void the_remux_chain_gates_then_dates() {
    Recorder rec;
    DurationFiller filler(rec);
    KeyframeGate gate(filler);
    PacketStage &head = gate;

    CHECK(head.open(nullptr, AVRational{1, 90000}) == 0, "open failed");
    CHECK(rec.opens_ == 1, "open did not reach the end of the chain");

    // A delta frame first, which must be dropped rather than dated.
    const bool keys[] = {false, true, false, false};
    for (int i = 0; i < 4; i++) {
        AVPacket *p = packet(i * 3000, keys[i]);
        CHECK(head.write(p) == 0, "write %d failed", i);
        free_packet(&p);
    }
    CHECK(gate.skipped() == 1, "skipped %llu, want 1",
          static_cast<unsigned long long>(gate.skipped()));

    CHECK(head.finish() == 0, "finish failed");
    CHECK(rec.seen_.size() == 3, "emitted %zu, want the 3 that survived the gate",
          rec.seen_.size());
    if (rec.seen_.size() == 3) {
        // The dropped packet must not have left a gap in the dating: the first
        // survivor is dated from the next survivor, not from what was dropped.
        CHECK(rec.seen_[0].pts == 3000, "the chain emitted %lld first",
              static_cast<long long>(rec.seen_[0].pts));
        for (size_t i = 0; i < 3; i++) {
            CHECK(rec.seen_[i].duration == 3000, "packet %zu dated %lld", i,
                  static_cast<long long>(rec.seen_[i].duration));
        }
    }
}

} // namespace

int main() {
    av_log_set_level(AV_LOG_ERROR);

    gate_drops_until_the_first_keyframe();
    gate_passes_everything_once_open();
    gate_reports_a_downstream_failure();

    filler_dates_each_packet_from_its_successor();
    filler_overwrites_the_one_tick_libav_gives_the_last_packet();
    filler_on_a_single_packet_emits_it();
    filler_dates_the_opening_access_unit();
    filler_drops_an_undated_packet_mid_stream();
    filler_finish_is_idempotent();

    the_remux_chain_gates_then_dates();

    if (failures == 0) {
        fprintf(stderr, "PASS\n");
        return 0;
    }
    fprintf(stderr, "%d check(s) failed\n", failures);
    return 1;
}
