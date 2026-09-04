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
# Every check the Go half of the project has to pass. Invoked identically by
# .circleci/config.yml and .github/workflows/ci.yml.

set -euo pipefail
source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"

# protoc and both plugins are needed by the generated-code check below: the gRPC
# description is the source of truth, so a stale checked-in codec is a failure.
require_tool go gofmt git protoc protoc-gen-go protoc-gen-go-grpc

tree_before="$(tree_state)"

cd "$REPO_ROOT/go"

log "toolchain"
go version

log "go mod download"
go mod download

# gofmt -l exits 0 whether or not it found anything, so the check has to look at
# the output.
log "gofmt"
unformatted="$(gofmt -l .)"
if [ -n "$unformatted" ]; then
    gofmt -d .
    die "these files are not gofmt'd:
$unformatted"
fi

# The generated codec must match api/hub.proto as committed.
#
# The CI image is the canonical generator: protoc's own version is stamped into
# the output, and the image's protoc has to stay matched to its libprotobuf for
# the C++ codegen to compile, so it cannot be pinned independently. Regenerate
# in the image -- see docker/ci/Dockerfile -- rather than with whatever protoc a
# workstation happens to have.
log "go generate reproduces the committed codec"
go generate ./...
git -C "$REPO_ROOT" diff --exit-code -- api/ go/api/ \
    || die "the generated code is stale, run 'go generate ./...' in go/"

log "go vet"
go vet ./...

# go install rather than go build: `go build ./...` drops one executable per main
# package into go/, which the tree-clean check below would then flag.
log "go install"
go install ./...

# -race because the whole design is concurrent: three agent kinds, a message bus
# and a media path. A run without it proves very little here.
log "go test -race"
go test -race -count=1 -timeout 15m ./...

# A stale go.mod or go.sum is a build that works on one machine and not the next.
log "go.mod is tidy"
go mod tidy
git -C "$REPO_ROOT" diff --exit-code -- go/go.mod go/go.sum \
    || die "go.mod or go.sum is not tidy, run 'go mod tidy' in go/"

log "the working tree is unchanged"
require_unchanged_tree "the Go build" "$tree_before"

log "go: OK"
