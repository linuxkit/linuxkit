#!/bin/sh

set -x

function failed {
	printf "cgroup not detected, suite FAILED\n" >&1
	exit 1
}

DEVICE=/dev/sdb

mount | grep cgroup2 || failed

stat /sys/fs/cgroup/newcgroup || failed

printf "cgroup2 detected, suite PASSED\n" >&1
