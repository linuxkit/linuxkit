#!/bin/sh

set -x

function failed {
	printf "blank volume test suite FAILED\n" >&1
	exit 1
}

# check that no files exist

[ -d /vola ] || failed

contents=$(ls -A /vola)
[ -z "$contents" ] || failed
printf "blank volume test suite PASSED\n" >&1
