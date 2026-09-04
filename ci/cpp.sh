#!/usr/bin/env bash
# Copyright (c) 2022-2024 The authors (see the AUTHORS file)
#
# This program is free software: you can redistribute it and/or modify
# it under the terms of the GNU Affero General Public License as
# published by the Free Software Foundation, either version 3 of the
# License, or (at your option) any later version.
#
# This program is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
# GNU Affero General Public License for more details.
#
# You should have received a copy of the GNU Affero General Public License
# along with this program.  If not, see <http://www.gnu.org/licenses/>.

#
# Configure, build and test the C++ half. Invoked identically by
# .circleci/config.yml and .github/workflows/ci.yml.

set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

require_tool cmake ctest pkg-config protoc grpc_cpp_plugin

tree_before="$(tree_state)"

BUILD_DIR="${BUILD_DIR:-$REPO_ROOT/cpp/build}"
BUILD_TYPE="${BUILD_TYPE:-Debug}"

log "dependencies"
pkg-config --modversion grpc++ protobuf libarchive libavformat libavcodec libavutil

# Out of source, and out of the tracked tree: the generated gRPC codec lands in
# the build directory, so a build never dirties the checkout.
log "cmake configure ($BUILD_TYPE)"
cmake -S "$REPO_ROOT/cpp" -B "$BUILD_DIR" -DCMAKE_BUILD_TYPE="$BUILD_TYPE"

log "cmake build"
cmake --build "$BUILD_DIR" --parallel "$(nproc --ignore=1 2>/dev/null || echo 2)"

log "ctest"
ctest --test-dir "$BUILD_DIR" --output-on-failure

log "the working tree is unchanged"
require_unchanged_tree "the C++ build" "$tree_before"

log "cpp: OK"
