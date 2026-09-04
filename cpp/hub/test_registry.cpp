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
// Admission of uploads, on its own.
//
// The registry is the one part of the upload path with real concurrency in it,
// and it has no gRPC in its interface on purpose, so it can be driven from a
// test with two threads and no server.
//

#include <atomic>
#include <chrono>
#include <cstdarg>
#include <cstdio>
#include <semaphore>
#include <thread>

#include "StreamRegistry.hpp"

namespace {

using namespace std::chrono_literals;

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

void a_free_key_is_admitted_and_given_back() {
    StreamRegistry reg;
    CHECK(reg.size() == 0, "a fresh registry holds %zu", reg.size());
    {
        const StreamRegistry::Lease lease(reg, "u/c", [] {});
        CHECK(lease.held(), "a free key was refused");
        CHECK(reg.size() == 1, "registry holds %zu, want 1", reg.size());
    }
    CHECK(reg.size() == 0, "the lease did not give the key back (%zu)", reg.size());
}

void different_keys_do_not_collide() {
    StreamRegistry reg;
    const StreamRegistry::Lease a(reg, "alice/door", [] {});
    const StreamRegistry::Lease b(reg, "bob/door", [] {});
    // Same camera name, different user: these are different streams.
    CHECK(a.held() && b.held(), "two distinct keys did not both fit");
    CHECK(reg.size() == 2, "registry holds %zu, want 2", reg.size());
}

// The case this exists for: the agent closes its upload and immediately opens
// another after a fresh RTSP session, so the newcomer must displace the
// incumbent instead of being turned away.
void the_newcomer_displaces_the_incumbent() {
    StreamRegistry reg(5s);

    std::binary_semaphore in_place{0};
    std::binary_semaphore asked_to_leave{0};
    std::atomic<int> cancels{0};

    std::thread incumbent([&] {
        const StreamRegistry::Lease first(reg, "u/c", [&] {
            cancels++;
            asked_to_leave.release();
        });
        CHECK(first.held(), "the incumbent was refused its own key");
        in_place.release();
        // Stand still until displaced, the way an upload blocked on a read
        // does; the lease is released as this scope ends.
        asked_to_leave.acquire();
    });

    in_place.acquire();

    const StreamRegistry::Lease second(reg, "u/c", [] {});
    CHECK(second.held(), "the newcomer was refused instead of displacing");
    CHECK(cancels.load() == 1, "the incumbent was asked to leave %d times",
          cancels.load());

    incumbent.join();
    CHECK(reg.size() == 1, "registry holds %zu after the handover, want 1",
          reg.size());
}

// An incumbent that ignores the cancellation must not be able to let a second
// writer into the same playlist.
void an_incumbent_that_will_not_leave_blocks_the_newcomer() {
    // A short wait, because this test spends it.
    StreamRegistry reg(50ms);

    const StreamRegistry::Lease squatter(reg, "u/c", [] {});
    CHECK(squatter.held(), "the squatter was refused its own key");

    const auto started = std::chrono::steady_clock::now();
    const StreamRegistry::Lease second(reg, "u/c", [] {});
    const auto waited = std::chrono::steady_clock::now() - started;

    CHECK(!second.held(), "a second writer was admitted to a held key");
    CHECK(waited >= 50ms, "gave up after %lldms, before the wait elapsed",
          static_cast<long long>(
                  std::chrono::duration_cast<std::chrono::milliseconds>(waited)
                          .count()));
    CHECK(reg.size() == 1, "registry holds %zu, want just the squatter",
          reg.size());
}

// A refused lease owns nothing, so falling out of scope must not evict the
// holder.
void a_refused_lease_releases_nothing() {
    StreamRegistry reg(20ms);

    const StreamRegistry::Lease squatter(reg, "u/c", [] {});
    {
        const StreamRegistry::Lease refused(reg, "u/c", [] {});
        CHECK(!refused.held(), "the second lease was admitted");
    }
    CHECK(reg.size() == 1, "the refused lease took the key with it (%zu)",
          reg.size());

    // And the holder is still the holder: a third attempt still has to wait.
    const StreamRegistry::Lease third(reg, "u/c", [] {});
    CHECK(!third.held(), "the key was free after a refused lease was destroyed");
}

} // namespace

int main() {
    a_free_key_is_admitted_and_given_back();
    different_keys_do_not_collide();
    the_newcomer_displaces_the_incumbent();
    an_incumbent_that_will_not_leave_blocks_the_newcomer();
    a_refused_lease_releases_nothing();

    if (failures == 0) {
        fprintf(stderr, "PASS\n");
        return 0;
    }
    fprintf(stderr, "%d check(s) failed\n", failures);
    return 1;
}
