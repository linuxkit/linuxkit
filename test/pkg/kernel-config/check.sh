#!/bin/sh

function failed {
	printf "Kernel config test suite FAILED\n"
	exit 1
}

/check-kernel-config.sh || failed

# Skip moby kernel checks on 5.x kernels for now.
# See: https://github.com/moby/moby/issues/38887
kernelVersion="$(uname -r)"
kernelMajor="${kernelVersion%%.*}"
if [ "$kernelMajor" -lt 5 ]; then
    bash /check-config.sh || failed
fi

printf "Kernel config test suite PASSED\n"
