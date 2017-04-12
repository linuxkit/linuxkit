#!/bin/sh

function failed {
	printf "Kernel config test suite FAILED\n"
}

/check-kernel-config.sh || failed
bash /check-config.sh || failed

printf "Kernel config test suite PASSED\n"

cat /etc/linuxkit
