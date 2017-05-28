#!/bin/sh

function failed {
	printf "dhcpcd test suite FAILED\n" >&1
	exit 1
}

LINK=$(iplink | grep eth0 | grep UP)
ADDR=$(echo `ifconfig eth0 2>/dev/null|awk '/inet addr:/ {print $2}'|sed 's/addr://'`)

[ -z "${LINK}" ] && failed
[ -z "${ADDR}" ] && failed

printf "dhcpcd test suite PASSED\n" >&1
