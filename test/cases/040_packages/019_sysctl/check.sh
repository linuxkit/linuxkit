#!/bin/sh

function failed {
	printf "sysctl test suite FAILED\n" >&1
	exit 1
}

# this is a non default value, so will fail if sysctl failed
[ "$(sysctl -n fs.inotify.max_user_watches)" -eq 524288 ] || failed

printf "sysctl test suite PASSED\n" >&1
