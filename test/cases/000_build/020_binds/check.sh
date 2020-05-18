#!/bin/sh

set -x

function failed {
	printf "bindmerge test suite FAILED\n" >&1
	exit 1
}

# the very fact that this is running means that the bind worked, so just need to check that the defaults also
# are there

[ -d /dev/mapper ] || failed
[ -d /hostroot ] || failed
printf "bindmerge test suite PASSED\n" >&1
