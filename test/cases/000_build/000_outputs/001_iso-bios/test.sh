#!/bin/sh
# SUMMARY: Check that iso-bios output format is generated
# LABELS:

set -e

# Source libraries. Uncomment if needed/defined
#. "${RT_LIB}"
. "${RT_PROJECT_ROOT}/_lib/lib.sh"

NAME=check

clean_up() {
	# remove any images
	find . -depth -iname "${NAME}*" -exec rm -rf {} \;
	rm -f test.yml
}

trap clean_up EXIT

moby build -format iso-bios -name "${NAME}" ../test.yml
[ -f "${NAME}.iso" ] || exit 1

exit 0
