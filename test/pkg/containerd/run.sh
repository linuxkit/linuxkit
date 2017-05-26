#!/bin/sh

function failed {
	printf "containerd test suite FAILED\n"
	exit 1
}

# unset -race (does not work on alpine; see golang/go#14481)
export TESTFLAGS=
make root-test || failed
printf "containerd test suite PASSED\n"
