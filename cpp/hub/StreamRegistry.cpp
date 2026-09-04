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

#include "StreamRegistry.hpp"

#include <utility>

StreamRegistry::Lease::Lease(StreamRegistry &registry, std::string key,
                             Canceller cancel)
        : registry_{registry}, key_{std::move(key)} {
    held_ = registry_.acquire(key_, std::move(cancel));
}

StreamRegistry::Lease::~Lease() {
    if (held_) {
        registry_.release(key_);
    }
}

std::size_t StreamRegistry::size() const {
    const std::lock_guard<std::mutex> guard(lock_);
    return active_.size();
}

bool StreamRegistry::acquire(const std::string &key, Canceller cancel) {
    std::unique_lock<std::mutex> guard(lock_);

    const auto it = active_.find(key);
    if (it != active_.end()) {
        if (it->second) {
            it->second();
        }
        // wait_for re-evaluates the predicate under the lock before it returns,
        // so a true answer means the key is free *and* still held by nobody
        // else: two newcomers racing cannot both proceed.
        const bool freed = freed_.wait_for(guard, eviction_wait_, [this, &key] {
            return active_.find(key) == active_.end();
        });
        if (!freed) {
            // It did not go. Refusing is the safe end of this: admitting a
            // second writer to one playlist is worse than losing an attempt,
            // and the agent retries.
            return false;
        }
    }

    active_.emplace(key, std::move(cancel));
    return true;
}

void StreamRegistry::release(const std::string &key) {
    {
        const std::lock_guard<std::mutex> guard(lock_);
        active_.erase(key);
    }
    // Notified outside the lock: a waiter woken inside it would only block
    // again on the way out.
    freed_.notify_all();
}
