project(enroute)
cmake_minimum_required(VERSION 3.10)

set_property(GLOBAL     PROPERTY    GLOBAL_DEPENDS_DEBUG_MODE   1)
SET(CMAKE_VERBOSE_MAKEFILE ON)

message("PROJECT_SOURCE_DIR         [${PROJECT_SOURCE_DIR}]")
message("PROJECT_BINARY_DIR         [${PROJECT_BINARY_DIR}]")
message("CMAKE_SOURCE_DIR           [${CMAKE_SOURCE_DIR}]")
message("CMAKE_BINARY_DIR           [${CMAKE_BINARY_DIR}]")
message("CMAKE_CURRENT_SOURCE_DIR   [${CMAKE_CURRENT_SOURCE_DIR}]")
message("CMAKE_CURRENT_BINARY_DIR   [${CMAKE_CURRENT_BINARY_DIR}]")

string(TIMESTAMP timestamp    %Y-%m-%d-%H-%M-%s) 
set(VERSION_BUILD_TIME ${timestamp})

#Find git, get last commit, save it in GIT_REVISION
find_package(Git)

if(GIT_FOUND)
    message("Found Git at:              [${GIT_EXECUTABLE}]")
        execute_process(
            COMMAND ${GIT_EXECUTABLE} rev-list --max-count=1 HEAD
            OUTPUT_VARIABLE GIT_REVISION
            ERROR_QUIET
        )
    if(NOT ${GIT_REVISION} STREQUAL "")
        string(STRIP ${GIT_REVISION} GIT_REVISION)
    endif()
    message(STATUS "Current revision at: [${GIT_REVISION}]")
else()
    message(SEND_ERROR    "Git not found. Please install git")
endif()

add_subdirectory(enroute-dp)
add_subdirectory(enroute-cp)
