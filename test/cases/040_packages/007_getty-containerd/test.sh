#!/bin/sh
# SUMMARY: Check that ctr can run containers
# LABELS:
# REPEAT:

set -e

# Source libraries. Uncomment if needed/defined
#. "${RT_LIB}"
. "${RT_PROJECT_ROOT}/_lib/lib.sh"

NAME=test-ctr

clean_up() {
	find . -depth -iname "test-ctr*" -not -iname "*.yml" -exec rm -rf {} \;
}
trap clean_up EXIT

moby build "${NAME}.yml"
[ -f "${NAME}-kernel" ] || exit 1
[ -f "${NAME}-initrd.img" ] || exit 1
[ -f "${NAME}-cmdline" ]|| exit 1
./test.exp
exit 0
