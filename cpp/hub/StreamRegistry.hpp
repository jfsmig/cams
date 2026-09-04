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
// Who is allowed to upload a given stream.
//

#pragma once

#include <chrono>
#include <condition_variable>
#include <cstddef>
#include <functional>
#include <map>
#include <mutex>
#include <string>

#include "Uncopyable.hpp"

// StreamRegistry admits one upload at a time per stream, and evicts the
// incumbent rather than refusing the newcomer.
//
// Two uploads of one camera would open the same directory and the same
// playlist, interleaving fragments from two unrelated RTP clocks into one
// timeline. The muxer would not complain and nothing downstream could untangle
// it, so exactly one has to be admitted.
//
// **The newcomer wins**, because of what a second upload means. The agent opens
// one only after establishing a fresh RTSP session, and it closes the previous
// one first -- see grpcMediaSink.onFrame, whose comment is "whatever came
// before is over". So by the time a second upload arrives, the incumbent is
// finished whether its own thread has noticed or not: the camera it was reading
// has gone. Refusing the newcomer would discard a live stream in favour of a
// dead one, and would cost the agent a retry period on every reconnect --
// including the ordinary one after a camera reboot.
class StreamRegistry : Uncopyable {
public:
    // How an incumbent is asked to leave. Called while the registry's lock is
    // held, so it must not block: it is expected to be an asynchronous
    // cancellation, and the incumbent needs that same lock to give the key
    // back.
    using Canceller = std::function<void()>;

    StreamRegistry() = default;

    explicit StreamRegistry(std::chrono::milliseconds eviction_wait)
            : eviction_wait_{eviction_wait} {}

    // Lease holds one key for as long as it is alive.
    class Lease : Uncopyable {
    public:
        Lease() = delete;

        // cancel is what a later Lease on the same key will call to displace
        // this one.
        Lease(StreamRegistry &registry, std::string key, Canceller cancel);

        ~Lease();

        // Whether the key was taken. A lease that was refused releases
        // nothing, so it is safe to let it fall out of scope.
        [[nodiscard]] bool held() const { return held_; }

    private:
        StreamRegistry &registry_;
        std::string key_;
        bool held_{false};
    };

    // How many streams are admitted. For diagnostics and for the tests.
    [[nodiscard]] std::size_t size() const;

private:
    [[nodiscard]] bool acquire(const std::string &key, Canceller cancel);

    void release(const std::string &key);

    mutable std::mutex lock_;
    std::condition_variable freed_;
    std::map<std::string, Canceller> active_;

    // How long to wait for an evicted incumbent to unwind. It has to write a
    // trailer on the way out, which touches the filesystem, so this is
    // generous rather than tight.
    std::chrono::milliseconds eviction_wait_{std::chrono::seconds(5)};
};
