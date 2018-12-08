#!/bin/sh

nsenter -t1 -m -- /usr/bin/bpftrace -l
if [ "$?" -ne "0" ]; then
	printf "bpftrace test suite FAILED\n" >&1
	exit 1
fi;

printf "bpftrace test suite PASSED\n" >&1