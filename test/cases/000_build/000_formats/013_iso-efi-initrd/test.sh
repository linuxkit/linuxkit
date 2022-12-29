#!/bin/sh
# SUMMARY: Check that iso-efi-initrd output format is generated
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

linuxkit build --format iso-efi-initrd --name "${NAME}" ../test.yml
[ -f "${NAME}"-efi-initrd.iso ] || exit 1

exit 0
