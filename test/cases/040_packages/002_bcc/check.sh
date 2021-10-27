#!/bin/sh

nsenter -t 1 -m --  /bin/ash -c "/usr/share/bcc/tools/softirqs 1 1"
if [ "$?" -ne "0" ]; then
	printf "bcc test suite FAILED\n" >&1
	exit 1
fi;

printf "bcc test suite PASSED\n" >&1
