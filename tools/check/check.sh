#!/bin/sh

function failed {
	printf "Moby test suite FAILED\n"
	/sbin/poweroff -f
}

/check-kernel-config.sh || failed
bash /check-config.sh || failed

printf "Moby test suite PASSED\n"

cat /etc/moby

/sbin/poweroff -f
