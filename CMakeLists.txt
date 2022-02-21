project(enroute)
cmake_minimum_required(VERSION 3.10)

set_property(GLOBAL PROPERTY GLOBAL_DEPENDS_DEBUG_MODE 1)
set(CMAKE_VERBOSE_MAKEFILE ON)

message("PROJECT_SOURCE_DIR         [${PROJECT_SOURCE_DIR}]")
message("PROJECT_BINARY_DIR         [${PROJECT_BINARY_DIR}]")
message("CMAKE_SOURCE_DIR           [${CMAKE_SOURCE_DIR}]")
message("CMAKE_BINARY_DIR           [${CMAKE_BINARY_DIR}]")
message("CMAKE_CURRENT_SOURCE_DIR   [${CMAKE_CURRENT_SOURCE_DIR}]")
message("CMAKE_CURRENT_BINARY_DIR   [${CMAKE_CURRENT_BINARY_DIR}]")

string(TIMESTAMP timestamp %Y-%m-%d-%H-%M-%s)
set(VERSION_BUILD_TIME ${timestamp})

# Find git, get last commit, save it in GIT_REVISION
find_package(Git)

if(GIT_FOUND)
  message("Found Git at:              [${GIT_EXECUTABLE}]")
  execute_process(
    COMMAND ${GIT_EXECUTABLE} rev-list --max-count=1 HEAD
    OUTPUT_VARIABLE GIT_REVISION
    ERROR_QUIET)
  if(NOT ${GIT_REVISION} STREQUAL "")
    string(STRIP ${GIT_REVISION} GIT_REVISION)
  endif()
  message(STATUS "Current revision at: [${GIT_REVISION}]")
else()
  message(SEND_ERROR "Git not found. Please install git")
endif()

# set(image_registry "gcr.io/enroute-10102020")
set(image_registry "saarasio")

add_custom_target(
  gofmtmodified
  COMMAND
    echo
    "git status | grep modified | grep -E '*.go$' | cut -d ' ' -f 4 | xargs -exec gofmt -l -s -w"
  VERBATIM)

add_subdirectory(enroutectl)
add_subdirectory(enroute-dp)
add_subdirectory(enroute-cp)
add_subdirectory(examples)
