#!/bin/sh

set -x

function failed {
	printf "rw_on_ro test suite FAILED\n" >&1
	exit 1
}

# this should fail as it is read-only
echo -n "file" > /vola/mytestfile || true

[ -e /vola/mytestfile ] && failed

printf "rw_on_ro test suite PASSED\n" >&1
