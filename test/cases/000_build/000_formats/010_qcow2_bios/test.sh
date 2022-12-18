#!/bin/sh
# SUMMARY: Check that qcow2 output format is generated
# LABELS: amd64

set -e

# Source libraries. Uncomment if needed/defined
#. "${RT_LIB}"
. "${RT_PROJECT_ROOT}/_lib/lib.sh"

NAME=check

clean_up() {
	rm -f ${NAME}*
}

trap clean_up EXIT

linuxkit build --format qcow2-bios --name "${NAME}" ../test.yml
[ -f "${NAME}.qcow2" ] || exit 1

exit 0
