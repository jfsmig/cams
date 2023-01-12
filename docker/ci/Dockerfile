ARG UBUNTU_VERSION=22.04

FROM ubuntu:${UBUNTU_VERSION} AS stage0

USER 0
RUN set -ex \
 && echo "Etc/UTC" > /etc/timezone \
 && ln -s /usr/share/zoneinfo/Etc/UTC /etc/localtime

# Install the system dependencies of the installation process
RUN set -ex \
 && apt-get update \
 && apt-get install -y --no-install-recommends \
 && apt-get autoremove --purge \
 && apt-get dist-upgrade -y \
 && apt-get install -y --no-install-recommends \
        ca-certificates \
        gosu \
        git \
        curl \
        build-essential \
        autoconf \
        libtool \
        pkg-config \
        cmake \
        clang \
        libc++-dev \
        zlib1g zlib1g-dev \
        libre2-9 libre2-dev \
        libssl-dev \
        ffmpeg \
        libavcodec58 libavcodec-dev \
        libavdevice58 libavdevice-dev \
        libavfilter7 libavfilter-dev \
        libavformat58 libavformat-dev \
        libavutil56 libavutil-dev \
 && update-ca-certificates \
 && apt-get clean \
 && rm -rf /var/lib/apt/lists/*

#----------------------------------------------------------------------------#
# Install the custom build tools

ARG CMAKE_VERSION=v3.25.1
RUN set -ex \
 && git clone https://github.com/Kitware/CMake.git \
 && cd CMake \
 && git checkout -b release-${CMAKE_VERSION} ${CMAKE_VERSION} \
 && git submodule update --init \
 && cmake -DCMAKE_BUILD_TYPE=Release -DCMAKE_INSTALL_PREFIX=/usr/local . \
 && make -j `nproc --ignore=2` \
 && make install \
 && cd - \
 && rm -rf CMake \
 && apt-get remove --purge -y cmake

ARG GRPC_VERSION=v1.51.1
RUN set -ex \
 && git clone https://github.com/grpc/grpc.git \
 && cd grpc \
 && git checkout -b release-${GRPC_VERSION} ${GRPC_VERSION} \
 && git submodule update --init \
 && cmake -DCMAKE_BUILD_TYPE=Release -DCMAKE_INSTALL_PREFIX=/usr/local . \
 && make -j `nproc --ignore=2` \
 && make install \
 && cd - \
 && rm -rf grpc

ARG GO_VERSION=1.19.2
ARG GO_URL=https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz
RUN set -ex\
 && curl  -K - -SL ${GO_URL} -o /tmp/go.linux-amd64.tar.gz \
 && tar -C /usr/local -zxf /tmp/go.linux-amd64.tar.gz \
 && update-alternatives --install /usr/bin/go    go    /usr/local/go/bin/go    100 \
 && update-alternatives --install /usr/bin/gofmt gofmt /usr/local/go/bin/gofmt 100 \
 && rm -f /tmp/go.linux-amd64.tar.gz
ENV GOPATH=/home/dev/go GOROOT=/usr/local/go

#----------------------------------------------------------------------------#
# Create an environment for a non-privileged user

ARG LOCAL_USER=dev

RUN set -ex \
 && mkdir /etc/skel/{build,go,src} \
 && groupadd -g 500 $LOCAL_USER \
 && useradd -g 500 -u 500 -m $LOCAL_USER \
 && usermod -aG sudo $LOCAL_USER \
 && usermod --shell /bin/bash $LOCAL_USER \
 && echo "export GIT_TRACE_PACKET=1" >> /home/$LOCAL_USER/.bashrc \
 && echo "export GIT_TRACE=1" >> /home/$LOCAL_USER/.bashrc \
 && echo "export GIT_CURL_VERBOSE=1" >> /home/$LOCAL_USER/.bashrc \
 && mkdir -p -m 700 /home/$LOCAL_USER/.ssh/ \
 && install -m 600 /dev/null /home/$LOCAL_USER/.ssh/config \
 && chown -R $LOCAL_USER:$LOCAL_USER /home/$LOCAL_USER/

USER ${LOCAL_USER}
WORKDIR /home/${LOCAL_USER}

RUN set -ex \
 && /usr/bin/go install golang.org/x/tools/cmd/goimports@latest \
 && /usr/bin/go install golang.org/x/tools/cmd/stringer@latest \
 && /usr/bin/go install honnef.co/go/tools/cmd/staticcheck@latest \
 && /usr/bin/go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28 \
 && /usr/bin/go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2

USER 0
WORKDIR /root
