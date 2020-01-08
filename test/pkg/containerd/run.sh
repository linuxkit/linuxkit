#!/bin/sh

function failed {
	printf "containerd test suite FAILED\n"
	exit 1
}

# Get these into the logs.
git describe HEAD
git rev-parse HEAD

# The unit tests need user_xattr support, which /tmp (a tmpfs) does not support.
mkdir -p /var/lib/tmp
export TMPDIR=/var/lib/tmp

# unset -race (does not work on alpine; see golang/go#14481)
export TESTFLAGS=
# disable devmapper tests
export SKIPTESTS="github.com/containerd/containerd/snapshots/devmapper github.com/containerd/containerd/snapshots/devmapper/dmsetup github.com/containerd/containerd/snapshots/devmapper/losetup"
make root-test || failed
printf "containerd test suite PASSED\n"
