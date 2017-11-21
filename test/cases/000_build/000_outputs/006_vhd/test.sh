#!/bin/sh
# SUMMARY: Check that vhd format is generated
# LABELS: skip
# VHD currently requires a lot of memory, disable for now

set -e

# Source libraries. Uncomment if needed/defined
#. "${RT_LIB}"
. "${RT_PROJECT_ROOT}/_lib/lib.sh"

NAME=check

clean_up() {
	rm -f ${NAME}*
}

trap clean_up EXIT

linuxkit build -format vhd -name "${NAME}" ../test.yml
[ -f "${NAME}.vhd" ] || exit 1

exit 0
