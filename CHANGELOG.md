# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/).

## [v0.8] - 2020-05-10
### Added

- Removed dependency on external `notary` and `manifest-tool` binaries for package builds
- Risc-V support for `binfmt`
- Support for GPT partitions
- `metadata` package support for Digital Ocean and Hetzner as well as loading from a file
- Support for `/sys/fs/bpf` in `init`
- Github Actions for CI

### Changed
- `alpine` base updated to 3.11
- `containerd` updated to v1.3.4
- `runc` updated to v1.0.0-rc9
- `binfmt` updated to qemu 4.2
- `node_exporter` updated to  v0.18.1
- `cadvisor` updated to v0.36.0
- WireGuard updated to 1.0.20200319
- Improved CDROM support and fixes to GCP and Scaleway providers in the `metadata` package
- Improved creation of `swap` files
- Improved RPI3 build

### Removed
- Containerized `qemu`
- Windows binary from release

## [v0.7] - 2019-04-17
### Added
- Reproducible `linuxkit build` for some output formats
- Support uncompressed kernels, e.g., for crosvm and firecracker.
- Support encrypted disks via `dm-crypt`
- New `bpftrace` package
- Support for USB devices in `qemu`

### Changed
- Alpine base updated to 3.9
- `containerd` updated to v1.2.6
- WireGuard updated to 0.0.20190227
- Updated Docker base API level
- VirtualBox improvements (multiple drives and network adapters)
- Fixed Windows path handling in `linuxkit`
- GCP: Improve error checking/handling

### Removed


## [v0.6] - 2018-07-26
### Added
- `linuxkit build` now works with private repositories and registries.
- `linuxkit pkg build` can build packages with sources outside the package directory.
- New `kernel+iso` format for `linuxkit build`.

### Changed
- `containerd` updated to v1.1.2.
- WireGuard updated to 0.0.20180718.
- Fixed SSH key handling on GCP.
- Changed name of logfiles when memlogd/logwrite is used.
- `moby/tool` code merged back into `linuxkit/linuxkit`
- Smaller `mkimage-*` packages.

### Removed



## [v0.5] - 2018-07-10
### Added
- New logging support with log rotation.
- Scaleway provider.
- Support for v4.17.x kernels.
- Kernel source are now included in the kernel packages.
- Improved documentation about debugging LinuxKit.

### Changed
- Switched to Alpine Linux 3.8 as the base.
- `containerd` updated to v1.1.1.
- `pkg/cadvisor` updated to v0.30.2
- `pkg/node_exporter` updated to 0.16.0
- WireGuard updated to 0.0.20180708.
- Linux firmware binaries update to latest.
- Improved support for building on Windows.
- Improved support for AWS/GCP metadata.
- Better handling of reboot/poweroff.

### Removed
- Support for v4.16.x. kernels as they have been EOLed.


## [v0.4] - 2018-05-12
### Added
- Support for v4.16.x kernels.
- Support for MPLS, USB_STORAGE, and SCTP support in the kernel config.
- Support for creating and booting from squashfs root filesystems.
- Super experimental support for crosvm.
- Support for compiling with go 1.10.
- Adjusted hyperkit support to be compatible with soon to be released Docker for Mac changes.

### Changed
- `containerd` updated to v1.1.0.
- WireGuard updated to 0.0.20180420.
- Intel CPU microcode update to 20180425.

### Removed
- Support for v4.15.x. kernels as they have been EOLed.
- `perf` support for 4.9.x kernels (the compile broke).


## [v0.3] - 2018-04-05
### Added
- Initial `s390x` support.
- Support for RealTime Linux kernels (`-rt`) on `x86_64` and `arm64`.
- Support for booting of `qcow2` disks via EFI.
- Support for CEPH filesystems in the kernel.
- Logging for `onboot` containers to `/var/log`
- Changelog file.

### Changed
- Switched the default kernel to 4.14.x.
- Update to `containerd` v1.0.3.
- Update to `notary` v0.6.0.
- Update WireGuard to 0.0.20180304.

### Removed
- Removed support for 4.4.x and 4.9.x kernels for `arm64`.


## [v0.2] - 2018-01-25
- Almost everything


## [v0.1] - 2017-??-??
- Sometime in 2017 we did a mini v0.1 release but we seem to have lost any trace of it :)


## [v0.0] - 2017-04-18
- Initial open sourcing of LinuxKit
