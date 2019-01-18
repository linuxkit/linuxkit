#!/bin/sh

function failed {
	printf "losetup test suite FAILED\n" >&1
	exit 1
}

LOOPFILE=$(losetup /dev/loop0 2>/dev/null | cut -d' ' -f3)

[ "$LOOPFILE" = "/var/test.img" ] || failed

printf "losetup test suite PASSED\n" >&1
