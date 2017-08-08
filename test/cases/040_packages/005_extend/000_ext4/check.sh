#!/bin/sh

set -x

function failed {
	printf "extend test suite FAILED\n" >&1
	exit 1
}

[ -f /var/lib/docker/bar ] || failed
touch /var/lib/docker/foo || failed
df -h | grep -qE "[4-5][0-9]{2}\.[0-9]{1,}M" || failed
printf "extend test suite PASSED\n" >&1
