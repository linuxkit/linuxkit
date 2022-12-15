#!/bin/sh
# SUMMARY: Check that ctr can run containers
# Test is amd64 specific
# LABELS: amd64
# REPEAT:

set -e

# Source libraries. Uncomment if needed/defined
#. "${RT_LIB}"
. "${RT_PROJECT_ROOT}/_lib/lib.sh"

NAME=test-ctr

clean_up() {
	rm -rf ${NAME}-*
}
trap clean_up EXIT

linuxkit build --format kernel+initrd --name "${NAME}" test.yml
[ -f "${NAME}-kernel" ] || exit 1
[ -f "${NAME}-initrd.img" ] || exit 1
[ -f "${NAME}-cmdline" ]|| exit 1
./test.exp
exit 0
