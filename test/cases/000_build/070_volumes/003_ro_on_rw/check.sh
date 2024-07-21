#!/bin/sh

set -x

function failed {
	printf "ro_on_rw test suite FAILED\n" >&1
	exit 1
}

# the file should not exist, as it was mounted read-only

[ -d /vola ] || failed
[ -e /vola/mytestfile ] && failed
printf "ro_on_rw test suite PASSED\n" >&1
