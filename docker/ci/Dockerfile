ARG UBUNTU_VERSION=22.04

FROM ubuntu:${UBUNTU_VERSION} AS stage0

USER 0
LABEL \
    maintainer="jf.smigielski@gmail.com" \
    version="0.1" \
    description="cams CI image, packed with all the deps"

ENV DEBIAN_FRONTEND=noninteractive

RUN set -ex \
 && echo "Etc/UTC" > /etc/timezone \
 && ln -s /usr/share/zoneinfo/Etc/UTC /etc/localtime

# Install the system dependencies of the installation process
RUN set -ex \
 && apt-get update \
 && apt-get dist-upgrade -y \
 && apt-get install -y --no-install-recommends \
        ca-certificates \
        gosu sudo \
        git \
        curl \
        build-essential \
        autoconf \
        libtool \
        pkg-config \
        gdb \
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
 && apt-get install -y --no-install-recommends \
        libmysofa-dev libmysofa1 \
        libvidstab-dev libvidstab1.1 \
        libva-dev libva2 \
        libass-dev libass9 \
        libpocketsphinx-dev libpocketsphinx3 \
        libsphinxbase-dev libsphinxbase3 \
        libbs2b-dev libbs2b0 \
        liblilv-dev liblilv-0-0 \
        libsratom-dev libsratom-0-0 \
        libsord-dev libsord-0-0 \
        libserd-dev libserd-0-0 \
        librubberband-dev librubberband2 \
        libsamplerate0-dev libsamplerate0 \
        flite1-dev libflite1 \
        libzmq3-dev libzmq5 \
        libzimg-dev libzimg2 \
        libmfx-dev libmfx1 \
        opencl-c-headers opencl-headers opencl-clhpp-headers nvidia-opencl-dev \
        libbz2-dev libbz2-1.0 \
        libgme-dev libgme0 \
        libopenmpt-dev libopenmpt0 \
        libchromaprint-dev libchromaprint1 \
        libbluray-dev libbluray2 \
        libgnutls28-dev libgnutls30 \
        librabbitmq-dev librabbitmq4 \
        libsrt-gnutls-dev \
        libssh-dev libssh-4 \
        libvpx-dev libvpx7 \
        libwebp-dev libwebp7 \
        libwebpmux3 \
        liblzma-dev liblzma5 \
        libdav1d-dev libdav1d5 \
        librsvg2-dev librsvg2-2 \
        libzvbi-dev libzvbi0 \
        libsnappy-dev libsnappy1v5 \
        libaom-dev libaom3 \
        libcodec2-dev libcodec2-1.0 \
        libgsm1-dev libgsm1 \
        libmp3lame-dev libmp3lame0 \
        libopenjp2-7-dev libopenjp2-7 \
        libopus-dev libopus0 \
        libtheora-dev libtheora0 \
        libspeex-dev libspeex1 \
        libshine-dev libshine3 \
        libtwolame-dev libtwolame0 \
        libx264-dev libx264-163 \
        libx265-dev libx265-199 \
        libxvidcore-dev libxvidcore4 \
        libsoxr-dev libsoxr-lsr0 \
        libvdpau-dev libvdpau1 \
        libdrm-dev libdrm2 \
        libxv-dev libxv1 \
 && update-ca-certificates \
 && apt-get autoremove --purge \
 && apt-get clean \
 && rm -rf /var/lib/apt/lists/*

#----------------------------------------------------------------------------#
# Install the custom build tools

ARG CMAKE_VERSION=v3.26.3
RUN set -ex \
 && git clone https://github.com/Kitware/CMake.git \
 && cd CMake \
 && git checkout -b release-${CMAKE_VERSION} ${CMAKE_VERSION} \
 && git submodule update --init \
 && ./bootstrap --prefix=/usr --parallel=`nproc --ignore=2` \
 && make -j `nproc --ignore=2` \
 && make install \
 && cd - \
 && rm -rf CMake

ARG GRPC_VERSION=v1.52.2
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

ARG GO_VERSION=1.20.4
ARG GO_URL=https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz
RUN set -ex\
 && curl  -K - -SL ${GO_URL} -o /tmp/go.linux-amd64.tar.gz \
 && tar -C /usr/local -zxf /tmp/go.linux-amd64.tar.gz \
 && update-alternatives --install /usr/bin/go    go    /usr/local/go/bin/go    100 \
 && update-alternatives --install /usr/bin/gofmt gofmt /usr/local/go/bin/gofmt 100 \
 && rm -f /tmp/go.linux-amd64.tar.gz

#----------------------------------------------------------------------------#
# Create an environment for a non-privileged user

RUN set -ex \
 && export GOBIN=/usr/bin \
 && /usr/bin/go install golang.org/x/tools/cmd/goimports@latest \
 && /usr/bin/go install golang.org/x/tools/cmd/stringer@latest \
 && /usr/bin/go install honnef.co/go/tools/cmd/staticcheck@latest \
 && /usr/bin/go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28.1 \
 && /usr/bin/go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2

ARG LOCAL_USER=dev
ENV GOPATH=/home/$LOCAL_USER/go GOROOT=/usr/local/go

#	RUN set -ex \
#	 && mkdir /etc/skel/build /etc/skel/go /etc/skel/src \
#	 && groupadd -g 1000 $LOCAL_USER \
#	 && useradd -g 1000 -u 1000 -m $LOCAL_USER \
#	 && usermod -aG sudo $LOCAL_USER \
#	 && usermod --shell /bin/bash $LOCAL_USER \
#	 && echo "export GIT_TRACE_PACKET=1" >> /home/$LOCAL_USER/.bashrc \
#	 && echo "export GIT_TRACE=1" >> /home/$LOCAL_USER/.bashrc \
#	 && echo "export GIT_CURL_VERBOSE=1" >> /home/$LOCAL_USER/.bashrc \
#	 && echo "export PATH=\"/home/$LOCAL_USER/go/bin:\$PATH\"" >> /home/$LOCAL_USER/.bashrc \
#	 && echo "export GOPATH=\"/home/$LOCAL_USER/go\"" >> /home/$LOCAL_USER/.bashrc \
#	 && echo "export GOROOT=\"/usr/local/go\"" >> /home/$LOCAL_USER/.bashrc \
#	 && mkdir -p -m 700 /home/$LOCAL_USER/.ssh/ \
#	 && install -m 600 /dev/null /home/$LOCAL_USER/.ssh/config \
#	 && chown -R $LOCAL_USER:$LOCAL_USER /home/$LOCAL_USER/
#	USER ${LOCAL_USER}
#	WORKDIR /home/${LOCAL_USER}

CMD ["/usr/bin/bash", "-l"]

