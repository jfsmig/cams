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

# The hub's media plane: a gRPC server that turns the RTP/RTCP upstream into HLS.
#
# Included from cpp/CMakeLists.txt, which owns the project-wide settings.

find_package(PkgConfig REQUIRED)
find_package(Threads REQUIRED)

find_program(PROTOBUF_PROTOC protoc REQUIRED)
find_program(GRPC_CPP_PLUGIN_EXECUTABLE grpc_cpp_plugin REQUIRED)

pkg_check_modules(GRPCPP REQUIRED IMPORTED_TARGET grpc++)
pkg_check_modules(PROTOBUF REQUIRED IMPORTED_TARGET protobuf)
pkg_check_modules(ARCHIVE REQUIRED IMPORTED_TARGET libarchive)

# The floor is FFmpeg 7.0 (libavformat 61): that is the release where
# avio_alloc_context's write_packet callback became unconditionally const, which
# is the signature the RTP source needs.
#
# Three modules that used to be required are not: libavfilter, libswscale and
# libswresample are for filtering, scaling and resampling, and this pipeline
# remuxes without ever decoding a frame.
pkg_check_modules(LIBAV REQUIRED IMPORTED_TARGET
    libavformat>=61
    libavcodec>=61
    libavutil>=59
)

# The gRPC description is the source of truth (see AGENTS.md); it lives at the
# top of the repository and is shared with the Go half. Anchored to this
# listfile rather than to CMAKE_SOURCE_DIR so the path does not depend on where
# cmake was invoked.
set(CAMS_API_DIR "${CMAKE_CURRENT_SOURCE_DIR}/../../api")
set(CAMS_CAPTURE "${CMAKE_CURRENT_SOURCE_DIR}/../../cams-capture-879216526.tar")

# Generated into the build tree, never into the source tree: the output of
# protoc-gen-grpc-cpp is coupled to the installed gRPC, so a committed copy is
# guaranteed to disagree with some other machine's toolchain.
set(PB_SOURCES
    "${CMAKE_CURRENT_BINARY_DIR}/hub.pb.h"
    "${CMAKE_CURRENT_BINARY_DIR}/hub.pb.cc"
    "${CMAKE_CURRENT_BINARY_DIR}/hub.grpc.pb.h"
    "${CMAKE_CURRENT_BINARY_DIR}/hub.grpc.pb.cc")

add_custom_command(
      OUTPUT ${PB_SOURCES}
      COMMAND ${PROTOBUF_PROTOC}
      ARGS --grpc_out "${CMAKE_CURRENT_BINARY_DIR}"
        --cpp_out "${CMAKE_CURRENT_BINARY_DIR}"
        -I "${CAMS_API_DIR}"
        "--plugin=protoc-gen-grpc=${GRPC_CPP_PLUGIN_EXECUTABLE}"
        "${CAMS_API_DIR}/hub.proto"
      DEPENDS "${CAMS_API_DIR}/hub.proto"
      COMMENT "Generating the C++ gRPC codec from api/hub.proto"
      VERBATIM)

add_library(alldeps INTERFACE)

# The pkg-config imported targets, rather than the *_LIBRARIES and
# *_STATIC_LIBRARIES variables the previous revision expanded by hand. Those
# expanded Libs.private too, so linking asked for every transitive dependency
# FFmpeg was built against -- x264, dav1d, gnutls, OpenCL and fifty more -- none
# of which is needed to link against a shared libavformat, and none of which the
# CI image installs.
target_link_libraries(alldeps INTERFACE
        PkgConfig::GRPCPP
        PkgConfig::PROTOBUF
        PkgConfig::LIBAV
        Threads::Threads)

target_compile_options(alldeps INTERFACE
        -g -pipe -Wall -Wextra
        $<$<CONFIG:DEBUG>:-Og -fno-inline>
        $<$<CONFIG:RELEASE>:-O2>)

add_library(baselib OBJECT
        ${PB_SOURCES}
        DurationFiller.cpp DurationFiller.hpp
        FragmentSink.cpp FragmentSink.hpp
        KeyframeGate.cpp KeyframeGate.hpp
        MediaPipeline.cpp MediaPipeline.hpp
        RtpSource.cpp RtpSource.hpp
        Sdp.cpp Sdp.hpp
        StreamRegistry.cpp StreamRegistry.hpp
        StreamStorage.cpp StreamStorage.hpp
        AvError.hpp FrameSource.hpp PacketStage.hpp Uncopyable.hpp)
target_link_libraries(baselib PUBLIC alldeps)
# PUBLIC so that everything linking baselib can reach the generated headers.
target_include_directories(baselib PUBLIC "${CMAKE_CURRENT_BINARY_DIR}")

add_executable(cams-rtp2hls
        MediaService.cpp MediaService.hpp
        rtp2hls.cpp)
target_link_libraries(cams-rtp2hls
        alldeps baselib)

add_executable(test-rtp2hls
        test_rtp2hls.cpp)
target_link_libraries(test-rtp2hls
        alldeps baselib
        PkgConfig::ARCHIVE)

add_test(NAME "rtp2hls" COMMAND test-rtp2hls "${CAMS_CAPTURE}")

# The stages on their own: no capture, no libarchive, no muxer.
add_executable(test-stages
        test_stages.cpp)
target_link_libraries(test-stages
        alldeps baselib)

add_test(NAME "stages" COMMAND test-stages)

# Upload admission, with two threads and no server.
add_executable(test-registry
        test_registry.cpp)
target_link_libraries(test-registry
        alldeps baselib)

add_test(NAME "registry" COMMAND test-registry)

# What survives a reconnect. Pure filesystem, no replay.
add_executable(test-storage
        test_storage.cpp)
target_link_libraries(test-storage
        alldeps baselib)

add_test(NAME "storage" COMMAND test-storage)

# The H.265 arm, against a stream the test builds with libx265 because no
# recorded HEVC capture exists. See test_hevc.cpp.
add_executable(test-hevc
        test_hevc.cpp)
target_link_libraries(test-hevc
        alldeps baselib)

add_test(NAME "hevc" COMMAND test-hevc)
