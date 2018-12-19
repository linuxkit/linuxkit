#!/bin/sh

function failed {
	printf "dm-crypt test suite FAILED\n" >&1
	exit 1
}

[ -b /dev/mapper/it_is_encrypted ] || failed

printf "dm-crypt test suite PASSED\n" >&1
