#!/bin/sh
# SUMMARY: Check that kernel+squashfs output format is generated
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

linuxkit build --format kernel+squashfs --name "${NAME}" ../test.yml
[ -f "${NAME}-kernel" ] || exit 1
[ -f "${NAME}-squashfs.img" ] || exit 1
[ -f "${NAME}-cmdline" ] || exit 1

exit 0
