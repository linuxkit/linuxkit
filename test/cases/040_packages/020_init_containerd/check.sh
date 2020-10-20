#!/bin/sh

function failed {
	printf "containerd commandline vars not set: FAILED\n" >/dev/console
	/sbin/poweroff -f
	exit 1
}

ps -ef | grep containerd | grep -q trace || failed

printf "containerd commandline vars test suite PASSED\n" >/dev/console

/sbin/poweroff -f
