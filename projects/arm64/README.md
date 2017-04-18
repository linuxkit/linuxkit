# ARM64 Moby

Attempting to build Moby for ARM64.


## Builder Container

`make -C alpine-build-kernel-arm64` builds the container used for building ARM64
images on ARM64.

Based off `aarch64/alpine:3.5`.

## Kernel Build

`make -C kernel-arm64` builds a container containing a compiled kernel with
associated modules and headers.

Configuration is constructed from `configs/`:
  * `arm64_defconfig`, the `arch/arm64/configs/defconfig` from Linux 4.9.15
  * `kernel_config` adding settings and overriding `CONFIG_BRIDGE m -> y`
  * `kernel_config.debug` if `$DEBUG != 0` configuring kernel debug options
