#!/bin/sh
# SUMMARY: Check that raw-efi output format is generated
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

linuxkit build --format raw-efi --name "${NAME}" ../test.yml
[ -f "${NAME}-efi.img" ] || exit 1

exit 0
