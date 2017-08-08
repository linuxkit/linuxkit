#!/bin/sh
# SUMMARY: Check that a btrfs partition can be extended
# LABELS:
# REPEAT:

set -ex

# Source libraries. Uncomment if needed/defined
#. "${RT_LIB}"
. "${RT_PROJECT_ROOT}/_lib/lib.sh"

NAME=test-extend-btrfs
DISK=disk0.img
clean_up() {
	find . -depth -iname "${NAME}*" -not -iname "*.yml" -exec rm -rf {} \;
	rm -rf "create*" || true
	rm -rf ${DISK} || true
}

trap clean_up EXIT

# Test code goes here
moby build --name create -output kernel+initrd test-create.yml
linuxkit run -disk file=${DISK},format=raw,size=256M create
[ -f ${DISK} ] || exit 1
# osx takes issue with bs=1M
dd if=/dev/zero bs=1048576 count=256 >> ${DISK}
moby build -name ${NAME} -output kernel+initrd test.yml
RESULT="$(linuxkit run -disk file=${DISK} ${NAME})"
echo "${RESULT}"
echo "${RESULT}" | grep -q "suite PASSED"

exit 0
