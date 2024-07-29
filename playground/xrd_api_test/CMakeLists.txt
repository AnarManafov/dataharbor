cmake_minimum_required(VERSION 3.5.0)

set(CMAKE_MODULE_PATH "${CMAKE_SOURCE_DIR}/cmake")

project(xrd_api_test VERSION 0.1.0 LANGUAGES C CXX)

message(STATUS "Setting support for c++17")
set (CMAKE_CXX_FLAGS "${CMAKE_CXX_FLAGS} -std=c++17")

find_package(xrootdclient REQUIRED)

add_library(XrdCl SHARED IMPORTED)
set_target_properties(XrdCl PROPERTIES
  IMPORTED_LOCATION "${XROOTD_XRDCL_LIB}"
  INTERFACE_INCLUDE_DIRECTORIES "${XROOTD_INCLUDE_DIR}"
)

add_executable(xrd_api_test main.cpp)

include_directories (${XROOTD_INCLUDE_DIR})

target_link_libraries (xrd_api_test
PUBLIC
XrdCl
)