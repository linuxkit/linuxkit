#!/bin/sh
# SUMMARY: Check that tar output format build is reproducible
# LABELS:

set -e

# Source libraries. Uncomment if needed/defined
#. "${RT_LIB}"
. "${RT_PROJECT_ROOT}/_lib/lib.sh"

NAME=check

clean_up() {
	rm -f ${NAME}*
}

trap clean_up EXIT

# -disable-content-trust to speed up the test
linuxkit build -disable-content-trust -format tar -name "${NAME}-1" ../test.yml
linuxkit build -disable-content-trust -format tar -name "${NAME}-2" ../test.yml

diff -q "${NAME}-1.tar" "${NAME}-2.tar" || exit 1

exit 0
