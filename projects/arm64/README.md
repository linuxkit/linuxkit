# ARM64 LinuxKit

Attempting to support ARM64 in LinuxKit.

Assume `CWD=$ROOT/projects/arm64` in what follows.

## Base

`make -C base` will build several base builder container images, based off
`aarch64/alpine:3.5`:
  * `alpine-build-kernel-arm64` builds the container used for building ARM64
    images on ARM64
  * `alpine-base-toybox`
  * `toybox-media`

## Tools

`make -C tools` will build various tools images, based off `aarch64/alpine:3.5`:
  * `go-compile` is the `go` toolchain used to build the `moby` command line
    tool

`make bin/moby` in `$ROOT` will then build the `moby` command line tool.

## Kernel Build

`make -C kernel-arm64` builds a container containing a compiled kernel with
associated modules and headers.

Configuration is constructed from `configs/`:
  * `arm64_defconfig`, the `arch/arm64/configs/defconfig` from Linux 4.9.15
  * `kernel_config` adding settings and overriding `CONFIG_BRIDGE m -> y`
  * `kernel_config.debug` if `$DEBUG != 0` configuring kernel debug options

## packet.net type 2A IPXE boot

  * iPXE boot only on first boot; subsequent boots assume disk boot will work,
    so need to `^B` into SOS session during boot to get to iPXE CLI
  * iPXE boot features are `DNS HTTP HTTPS iSCSI TFTP VLAN AoE EFI Menu` -- so
    `-kernel` and `-bzImage` `moby` outputs don't work as both are gzipped.

    ```
    mv foo-kernel foo-kernel.gz
    gnuzip foo-kernel.gz
    ```

    ...boots but panics because no root device
  * Same needed to be done with `initrd.img` as well as specifying
    `initrd=initrd.img` on kernel boot command line

# Notes

  * `Makefile` `find ... -depth ...` is broken on Linux
  * vendor path in `src/cmd/moby/build.go` refs `docker/moby`
  * Alpine `apk update ... && ...` preamble lines could be collapsed?
  * `aarch64/alpine:linux-headers` seems to be 4.4.6 but we build 4.9.x?
  * `runc` build with my `go-1.8` container, not `1.7`?
