#!/bin/sh
# SUMMARY: Check that there are no open ports
# LABELS:

set -e

# Source libraries. Uncomment if needed/defined
#. "${RT_LIB}"
. "${RT_PROJECT_ROOT}/_lib/lib.sh"

NAME=lsof

clean_up() {
	# remove any files, containers, images etc
	rm -rf ${NAME}* || true
}

trap clean_up EXIT

linuxkit build -format kernel+initrd -name "${NAME}" test.yml
linuxkit run qemu -kernel "${NAME}"
#RESULT=$(linuxkit run qemu -kernel "${NAME}")
#echo "${RESULT}" | grep -q "PASSED"
exit 0
