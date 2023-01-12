ARG UBUNTU_VERSION=22.04

FROM ubuntu:${UBUNTU_VERSION} AS stage0

USER 0

RUN set -eux \
 && apt-get update \
 && apt-get install -y --no-install-recommends \
 && apt-get autoremove --purge \
 && apt-get dist-upgrade -y \
 && apt-get install -y --no-install-recommends \
        ca-certificates \
        gosu \
        git \
        build-essential autoconf libtool pkg-config \
        cmake clang libc++-dev \
        zlib1g zlib1g-dev \
        libre2-9 libre2-dev \
        libssl-dev \
 && update-ca-certificates

RUN set -eux \
 && git clone https://github.com/Kitware/CMake.git \
 && git clone https://github.com/grpc/grpc.git

ARG CMAKE_VERSION=v3.25.1
RUN set -eux \
 && cd CMake \
 && git checkout -b release-${CMAKE_VERSION} ${CMAKE_VERSION} \
 && git submodule update --init \
 && cmake -DCMAKE_BUILD_TYPE=Release -DCMAKE_INSTALL_PREFIX=/opt/cmake .

RUN set -eux \
 && cd CMake \
 && make -j `nproc --ignore=2`

ARG GRPC_VERSION=v1.51.1
RUN set -eux \
 && cd grpc \
 && git checkout -b release-${GRPC_VERSION} ${GRPC_VERSION} \
 && git submodule update --init

RUN set -eux \
 && cd CMake \
 && make install \
 && apt-get remove --purge -y cmake

RUN set -eux \
 && cd grpc \
 && export PATH="/opt/cmake/bin:$PATH" \
 && cmake -DCMAKE_BUILD_TYPE=Release -DCMAKE_INSTALL_PREFIX=/opt/grpc .

RUN set -ex \
 && cd grpc \
 && export PATH="/opt/cmake/bin:$PATH" \
 && make -j `nproc --ignore=2`

