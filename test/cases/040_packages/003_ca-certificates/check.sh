#!/bin/sh

function failed {
	printf "ca-certificates test suite FAILED\n" >&1
	exit 1
}

[ -d /host-etc/ssl/ ] || failed
[ -d /host-etc/ssl/certs ] || failed
[ -f /host-etc/ssl/certs/ca-certificates.crt ] || failed

printf "ca-certificates test suite PASSED\n" >&1
