#!/bin/sh
# SUMMARY: Check that a btrfs partition can be extended
# LABELS:
# REPEAT:

set -ex

# Source libraries. Uncomment if needed/defined
#. "${RT_LIB}"
. "${RT_PROJECT_ROOT}/_lib/lib.sh"

NAME=test-extend
DISK=disk0.img
clean_up() {
	find . -depth -iname "${NAME}*" -not -iname "*.yml" -exec rm -rf {} \;
	rm -rf ${DISK} || true
	docker rmi ${NAME} || true
}

trap clean_up EXIT

# Test code goes here
rm -rf disk0.img || true
docker build -t ${NAME} .
moby build --name create -output kernel+initrd test-create.yml
linuxkit run -disk file=${DISK},size=256M create
rm -rf "create*"
[ -f ${DISK} ] || exit 1
docker run -i --rm --privileged -v "$PWD:/tmp" -w /tmp ${NAME} ./extend.sh ${DISK}
moby build -name ${NAME} -output kernel+initrd test.yml
RESULT="$(linuxkit run -disk file=${DISK} ${NAME})"
echo "${RESULT}"
echo "${RESULT}" | grep -q "suite PASSED"

exit 0
