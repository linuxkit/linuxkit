#!/bin/sh

function failed {
	printf "Kernel config test suite FAILED\n"
	exit 1
}

/check-kernel-config.sh || failed
bash /check-config.sh || failed

printf "Kernel config test suite PASSED\n"
