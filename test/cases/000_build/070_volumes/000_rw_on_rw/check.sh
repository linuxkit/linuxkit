#!/bin/sh

set -x

function failed {
	printf "rw_on_rw test suite FAILED\n" >&1
	exit 1
}

# check that the files we created are there and have the contents

[ -d /vola ] || failed
[ -e /vola/mytestfile ] || failed
[ "$(cat /vola/mytestfile)" == "file" ] || failed
printf "rw_on_rw test suite PASSED\n" >&1
