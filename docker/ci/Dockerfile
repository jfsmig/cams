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

# The CI image for both CI systems: .circleci/config.yml and
# .github/workflows/ci.yml run ci/go.sh and ci/cpp.sh inside it.
#
# Everything the build needs comes from the distribution, apart from the Go
# toolchain, which has to stay ahead of go/go.mod. The previous revision built
# CMake and gRPC from source and pinned ~55 of FFmpeg's own build dependencies
# under version-suffixed package names, which made the image unbuildable on any
# newer base — and it never installed libarchive, so the C++ half could not be
# built here at all.

ARG UBUNTU_VERSION=26.04

FROM ubuntu:${UBUNTU_VERSION} AS cams-dev

USER 0
LABEL \
    maintainer="jf.smigielski@gmail.com" \
    version="0.2" \
    description="Cams CI image, packed with all the build dependencies"

ENV DEBIAN_FRONTEND=noninteractive

RUN set -ex \
 && echo "Etc/UTC" > /etc/timezone \
 && ln -sf /usr/share/zoneinfo/Etc/UTC /etc/localtime

# The build dependencies, all unversioned: naming SONAME digits is what pinned
# the previous image to one release of the distribution.
#
# libav* is needed by cpp/hub, which requires FFmpeg >= 7 (libavformat 61).
# Only three modules are asked for: the media path remuxes and never decodes, so
# it has no use for libavfilter, libswscale or libswresample. Asking for less is
# not the same as excluding them -- the distribution pulls all three in behind
# these, headers included, so this list documents the intent and does not
# enforce it. What keeps a transcoder out of cpp/hub is review.
# libarchive by the tar replay harness in cpp/hub/test_rtp2hls.cpp; protoc and
# the two protoc plugins by both halves, since api/hub.proto is the source of
# truth for the Go codec as well.
RUN set -ex \
 && apt-get update \
 && apt-get dist-upgrade -y \
 && apt-get install -y --no-install-recommends \
        ca-certificates \
        curl \
        git \
        openssh-client \
        sudo \
        build-essential \
        clang \
        gdb \
        pkg-config \
        cmake \
        ninja-build \
        libssl-dev \
        zlib1g-dev \
        libgrpc++-dev \
        libprotobuf-dev \
        protobuf-compiler \
        protobuf-compiler-grpc \
        libarchive-dev \
        libavformat-dev \
        libavcodec-dev \
        libavutil-dev \
 && update-ca-certificates \
 && apt-get clean \
 && rm -rf /var/lib/apt/lists/*

#----------------------------------------------------------------------------#
# The Go toolchain, from upstream rather than from the distribution: it has to
# be at or above the version in go/go.mod.

ARG GO_VERSION=1.27.1
ARG GO_URL=https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz
RUN set -ex \
 && curl -SL ${GO_URL} -o /tmp/go.linux-amd64.tar.gz \
 && tar -C /usr/local -zxf /tmp/go.linux-amd64.tar.gz \
 && update-alternatives --install /usr/bin/go    go    /usr/local/go/bin/go    100 \
 && update-alternatives --install /usr/bin/gofmt gofmt /usr/local/go/bin/gofmt 100 \
 && rm -f /tmp/go.linux-amd64.tar.gz

# Installed into /usr/local/bin, as root, so that they are on PATH for whatever
# UID the CI runs as. ci/go.sh needs the two protoc plugins for its check that
# the committed codec still matches api/hub.proto.
#
# The two protoc plugins are pinned, not @latest, and pinned to the versions
# that match go/go.mod: protoc-gen-go tracks google.golang.org/protobuf and
# protoc-gen-go-grpc tracks google.golang.org/grpc. A newer plugin emits code
# against a newer runtime -- protoc-gen-go-grpc 1.6 wants
# grpc.SupportPackageIsVersion9, which grpc 1.61 does not have -- so @latest
# here would break the build every time the image was rebuilt. Bump these two
# together with go.mod, then regenerate and commit the codec.
ARG PROTOC_GEN_GO_VERSION=v1.32.0
ARG PROTOC_GEN_GO_GRPC_VERSION=v1.3.0
ENV GOROOT=/usr/local/go
RUN set -ex \
 && export GOBIN=/usr/local/bin GOPATH=/tmp/gopath \
 && for tool in \
        golang.org/x/tools/cmd/goimports@latest \
        golang.org/x/tools/cmd/stringer@latest \
        honnef.co/go/tools/cmd/staticcheck@latest \
        google.golang.org/protobuf/cmd/protoc-gen-go@${PROTOC_GEN_GO_VERSION} \
        google.golang.org/grpc/cmd/protoc-gen-go-grpc@${PROTOC_GEN_GO_GRPC_VERSION} \
    ; do /usr/local/go/bin/go install "$tool" ; done \
 && rm -rf /tmp/gopath

#----------------------------------------------------------------------------#
# Fail here, at image build time, rather than halfway through a CI run.

RUN set -ex \
 && pkg-config --atleast-version=61 libavformat \
 && pkg-config --atleast-version=61 libavcodec \
 && pkg-config --atleast-version=59 libavutil \
 && pkg-config --exists grpc++ \
 && pkg-config --exists protobuf \
 && pkg-config --exists libarchive \
 && for t in go gofmt cmake ctest pkg-config git \
             protoc grpc_cpp_plugin protoc-gen-go protoc-gen-go-grpc ; do \
        command -v "$t" >/dev/null || { echo "missing: $t" >&2 ; exit 1 ; } ; \
    done \
 && go version \
 && cmake --version | head -1 \
 && printf 'libavformat %s / grpc++ %s\n' \
        "$(pkg-config --modversion libavformat)" \
        "$(pkg-config --modversion grpc++)"

#----------------------------------------------------------------------------#
# An unprivileged user for interactive use. CI runs as root, which is why the
# tools above are in /usr/local/bin and not under this user's home.

ARG LOCAL_USER=dev
RUN set -ex \
 && useradd -U -m -s /bin/bash $LOCAL_USER

VOLUME /workspace

CMD ["/bin/bash", "-l"]
