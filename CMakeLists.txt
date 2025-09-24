cmake_minimum_required(VERSION 3.15)

project(atdtool)

set(CMAKE_MODULE_PATH ${CMAKE_MODULE_PATH} "${CMAKE_CURRENT_SOURCE_DIR}/cmake")
include(CMakeDetermineGoCompiler)

# installer directory
if (NOT PROJECT_INSTALL_BAS_DIR)
    set (PROJECT_INSTALL_BAS_DIR "${PROJECT_BINARY_DIR}/${PROJECT_BUILD_NAME}")
endif ()
if(NOT EXISTS ${PROJECT_INSTALL_BAS_DIR})
    file(MAKE_DIRECTORY ${PROJECT_INSTALL_BAS_DIR})
    message(STATUS "create ${PROJECT_INSTALL_BAS_DIR} for build target.")
endif()

if (CMAKE_Go_COMPILER)
    include(cmake/golang.cmake)
    include(cmake/flags.cmake)

    add_subdirectory(cmd/atdtool)
    add_subdirectory(cmd/logarchive)

endif()
