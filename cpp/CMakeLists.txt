cmake_minimum_required(VERSION 3.20)
project(cams-cpp)

set(CMAKE_CXX_STANDARD 20)
find_package(PkgConfig)
find_package (Threads)

find_program(PROTOBUF_PROTOC protoc)
find_program(GRPC_CPP_PLUGIN_EXECUTABLE grpc_cpp_plugin)

pkg_check_modules(GRPCPP REQUIRED grpc++)
pkg_check_modules(PROTOBUF REQUIRED protobuf)
pkg_check_modules(LIBAV REQUIRED IMPORTED_TARGET
    libavfilter
    libavformat
    libavcodec
    libswresample
    libswscale
    libavutil
)

add_custom_command(
      OUTPUT
        "${CMAKE_CURRENT_SOURCE_DIR}/hub.pb.h"
        "${CMAKE_CURRENT_SOURCE_DIR}/hub.pb.cc"
        "${CMAKE_CURRENT_SOURCE_DIR}/hub.grpc.pb.h"
        "${CMAKE_CURRENT_SOURCE_DIR}/hub.grpc.pb.cc"
      COMMAND ${PROTOBUF_PROTOC}
      ARGS --grpc_out "${CMAKE_CURRENT_SOURCE_DIR}"
        --cpp_out "${CMAKE_CURRENT_SOURCE_DIR}"
        -I "${CMAKE_SOURCE_DIR}/../protos"
        --plugin=protoc-gen-grpc="${GRPC_CPP_PLUGIN_EXECUTABLE}"
        "${CMAKE_SOURCE_DIR}/../protos/hub.proto"
      DEPENDS "${CMAKE_SOURCE_DIR}/../protos/hub.proto"
      COMMAND_EXPAND_LISTS)

add_executable(cams-rtp-to-hls
        "${CMAKE_CURRENT_SOURCE_DIR}/hub.pb.h"
        "${CMAKE_CURRENT_SOURCE_DIR}/hub.pb.cc"
        "${CMAKE_CURRENT_SOURCE_DIR}/hub.grpc.pb.h"
        "${CMAKE_CURRENT_SOURCE_DIR}/hub.grpc.pb.cc"
        main.cpp
        fmp4.cpp MediaService.cpp MediaService.hpp MediaDecoder.cpp MediaDecoder.hpp MediaEncoder.cpp MediaEncoder.hpp StreamStorage.cpp StreamStorage.hpp Uncopyable.hpp)

target_compile_options(cams-rtp-to-hls PUBLIC
        -g -pipe -Wall -Wextra
        $<$<CONFIG:DEBUG>:-Og -fno-inline>
        $<$<CONFIG:RELEASE>:-O2>)

target_include_directories(cams-rtp-to-hls PUBLIC
        ${LIBAV_INCLUDE_DIRS}
        ${GRPCPP_INCLUDE_DIRS}
        ${PROTOBUF_INCLUDE_DIRS}
)

target_link_directories(cams-rtp-to-hls PUBLIC
        ${LIBAV_LIBRARY_DIRS}
        ${GRPCPP_LIBRARY_DIRS}
        ${PROTOBUF_LIBRARY_DIRS}
)

target_link_libraries(cams-rtp-to-hls
        ${GRPCPP_LIBRARIES}
        ${GRPCPP_STATIC_LIBRARIES}
        ${PROTOBUF_LIBRARIES}
        ${PROTOBUF_STATIC_LIBRARIES}
        ${LIBAV_LIBRARIES}
        ${LIBAV_STATIC_LIBRARIES}
)
