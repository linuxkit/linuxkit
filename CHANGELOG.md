# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/).

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
