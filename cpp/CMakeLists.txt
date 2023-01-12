cmake_minimum_required(VERSION 3.20)
project(cams)

set(CMAKE_CXX_STANDARD 20)
find_package(PkgConfig)
find_package (Threads)

find_program(PROTOBUF_PROTOC protoc)
find_program(GRPC_CPP_PLUGIN_EXECUTABLE grpc_cpp_plugin)

pkg_check_modules(GRPCPP REQUIRED grpc++)

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
        fmp4.cpp)

target_include_directories(cams-rtp-to-hls PUBLIC
        ${GRPCPP_INCLUDE_DIRS})
target_link_directories(cams-rtp-to-hls PUBLIC
        ${GRPCPP_LIBRARY_DIRS})
target_link_libraries(cams-rtp-to-hls
        ${GRPCPP_LIBRARIES})