cmake_minimum_required(VERSION 3.20)
project(cams)

set(CMAKE_CXX_STANDARD 20)
find_package(PkgConfig)
find_package (Threads)

add_executable(fmp4 fmp4.cpp)