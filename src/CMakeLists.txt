cmake_minimum_required(VERSION 3.9.0 FATAL_ERROR)
project(fs CXX)

if(NOT CMAKE_BUILD_TYPE )
MESSAGE(STATUS "no build type defined, falling back to Debug" )
set(CMAKE_BUILD_TYPE Debug )
endif()

set(CMAKE_CXX_STANDARD 14)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_CXX_EXTENSIONS OFF)

set(CMAKE_CXX_FLAGS "${CMAKE_CXX_FLAGS} -Wall -pedantic -Wextra -D_FILE_OFFSET_BITS=64")
set(CMAKE_CXX_FLAGS_RELEASE "${CMAKE_CXX_FLAGS_RELEASE} -g -O3")
set(CMAKE_CXX_FLAGS_DEBUG "${CMAKE_CXX_FLAGS_DEBUG} -g3 -O0")

find_library(THREAD_LIBRARY NAMES pthread)
find_library(LOG_LIBRARY NAMES log4cpp)
find_library(CURL_LIBRARY NAMES curl)
find_library(HTTP_LIBRARY NAMES microhttpd)
find_library(ARCHIVE_LIBRARY NAMES archive)
find_library(FUSE_LIBRARY NAMES fuse)
find_library(CPPUNIT_LIBRARY NAMES cppunit)

file(GLOB_RECURSE SRC *.cpp)
list(FILTER SRC EXCLUDE REGEX "^.*/bin/.*$")

file(GLOB_RECURSE SRC_BIN bin/*.cpp)
file(GLOB_RECURSE SRC_UNIT_TESTS ../test/*.cpp)

add_library(flocons SHARED ${SRC})

set(LIBS
    flocons
    ${THREAD_LIBRARY}
    ${LOG_LIBRARY}
    ${CURL_LIBRARY}
    ${HTTP_LIBRARY}
    ${ARCHIVE_LIBRARY}
    ${FUSE_LIBRARY}
)

add_executable(unit-tests ${SRC_UNIT_TESTS})
target_link_libraries(unit-tests ${LIBS} ${CPPUNIT_LIBRARY})

enable_testing()
add_test(NAME unit_tests COMMAND ${PROJECT_BINARY_DIR}/unit-tests)

foreach(BIN IN ITEMS ${SRC_BIN})
get_filename_component(BIN_NAME ${BIN} NAME_WE)
add_executable(${BIN_NAME} ${BIN})
target_link_libraries(${BIN_NAME} ${LIBS})
endforeach(BIN)

configure_file(../conf/log4cpp.properties ${PROJECT_BINARY_DIR}/log4cpp.properties COPYONLY)