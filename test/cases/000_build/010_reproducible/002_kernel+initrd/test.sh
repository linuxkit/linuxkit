#!/bin/sh
# SUMMARY: Check that kernel+initrd output format build is reproducible
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
linuxkit build -disable-content-trust -format kernel+initrd -name "${NAME}-1" ../test.yml
linuxkit build -disable-content-trust -format kernel+initrd -name "${NAME}-2" ../test.yml

diff -q "${NAME}-1-cmdline"    "${NAME}-2-cmdline"    || exit 1
diff -q "${NAME}-1-kernel"     "${NAME}-2-kernel"     || exit 1
diff -q "${NAME}-1-initrd.img" "${NAME}-2-initrd.img" || exit 1

exit 0
