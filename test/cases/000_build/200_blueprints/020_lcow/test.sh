#!/bin/sh
# SUMMARY: Test the build of LCOW blueprint
# LABELS:

set -e

# Source libraries. Uncomment if needed/defined
#. "${RT_LIB}"
. "${RT_PROJECT_ROOT}/_lib/lib.sh"

NAME=lcow

clean_up() {
	rm -f ${NAME}-*
}

trap clean_up EXIT

# Test code goes here
moby build -format kernel+initrd -name "${NAME}" "${LINUXKIT_BLUEPRINTS_DIR}/lcow.yml" 
[ -f "${NAME}-kernel" ] || exit 1
[ -f "${NAME}-initrd.img" ] || exit 1
[ -f "${NAME}-cmdline" ] || exit 1

exit 0

