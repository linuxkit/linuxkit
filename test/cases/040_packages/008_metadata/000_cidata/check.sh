#!/bin/sh

set -x

function failed {
	printf "metadata test suite FAILED\n" >&1
	exit 1
}

DEVICE=/dev/sdb

[ -f /run/config/provider ] || failed
[ "$(cat /run/config/provider)" = "CDROM ${DEVICE}" ] || failed
[ -f /run/config/userdata ] || failed
grep -q supersecret /run/config/userdata || failed
printf "metadata test suite PASSED\n" >&1
