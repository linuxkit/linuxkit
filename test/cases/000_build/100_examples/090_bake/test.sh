#!/bin/sh
# SUMMARY: Test linuxkit bake
# LABELS:

set -e

# Source libraries. Uncomment if needed/defined
#. "${RT_LIB}"
. "${RT_PROJECT_ROOT}/_lib/lib.sh"
NAME=linuxkit_template
LINUXKIT_PKG_ROOT="${RT_PROJECT_ROOT}/../../pkg"
LINUXKIT_BAKED="${NAME}_baked.yml"

clean_up() {
	rm -f ${NAME}*
}

trap clean_up EXIT

# Test code goes here
linuxkit bake --pkgroot "${LINUXKIT_PKG_ROOT}" "${LINUXKIT_EXAMPLES_DIR}/${NAME}.yml" > "${LINUXKIT_BAKED}"
linuxkit build -name "${NAME}" "${LINUXKIT_BAKED}"

exit 0
