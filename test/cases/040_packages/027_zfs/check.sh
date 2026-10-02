#!/bin/sh

failed() {
	printf "zfs test suite FAILED\n"
	exit 1
}

modprobe zfs || failed
dmesg | grep -qi "ZFS:" || failed

IMG=/tmp/zfs-test.img
dd if=/dev/zero of="$IMG" bs=1M count=64 >/dev/null 2>&1 || failed
zpool create -m legacy ziptest "$IMG" || failed
zpool list ziptest | grep -q ziptest || failed
zfs list ziptest | grep -q ziptest || failed
zpool destroy ziptest || failed
rm -f "$IMG"

printf "zfs test suite PASSED\n"
