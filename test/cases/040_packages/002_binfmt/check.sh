#!/bin/sh

function failed {
	printf "binfmt test suite FAILED\n" >&1
	exit 1
}

[ -e /binfmt_misc/qemu-aarch64 ] || failed
[ -e /binfmt_misc/qemu-arm ]     || failed
[ -e /binfmt_misc/qemu-ppc64le ] || failed

printf "binfmt test suite PASSED\n" >&1
