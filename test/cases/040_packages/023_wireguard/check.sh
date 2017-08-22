#!/bin/sh

function success {
	printf "wireguard test suite PASSED\n" >&1
	exit 0
}

function failed {
	printf "wireguard test suite FAILED\n" >&1
	exit 1
}

if [ "$1" = "shutdown" ]
then
	[ -f /tmp/ok ] && success
	failed
	exit 0
fi

# Nginx may not be up immediately as service startup is async
for s in $(seq 1 10)
do
	wget -O - http://192.168.2.1/ && echo "success" > /tmp/ok && halt
	sleep 1
done

halt
