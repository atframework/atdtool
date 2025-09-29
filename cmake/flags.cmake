# Setting Paddle Compile Flags
include(CheckCXXCompilerFlag)
include(CheckCCompilerFlag)
include(CheckCXXSymbolExists)
include(CheckTypeSize)

set(CMAKE_CXX_STANDARD 20)
set(CMAKE_C_STANDARD 11)

# Common gpu architectures: Kepler, Maxwell
foreach(capability 30 35 50)
      list(APPEND __arch_flags " -gencode arch=compute_${capability},code=sm_${capability}")
endforeach()

if (CUDA_VERSION VERSION_GREATER "7.0" OR CUDA_VERSION VERSION_EQUAL "7.0")
      list(APPEND __arch_flags " -gencode arch=compute_52,code=sm_52")
endif()

# Modern gpu architectures: Pascal
if (CUDA_VERSION VERSION_GREATER "8.0" OR CUDA_VERSION VERSION_EQUAL "8.0")
      list(APPEND __arch_flags " -gencode arch=compute_60,code=sm_60")
endif()

set(CUDA_NVCC_FLAGS ${__arch_flags} ${CUDA_NVCC_FLAGS})