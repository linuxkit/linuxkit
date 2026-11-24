#!/bin/sh
# SUMMARY: Check that the zfs kernel module package works
# LABELS: skip
# REPEAT:

set -e

# Source libraries. Uncomment if needed/defined
#. "${RT_LIB}"
. "${RT_PROJECT_ROOT}/_lib/lib.sh"
NAME=zfs

clean_up() {
	docker rmi zfs-check-test || true
	rm -rf ${NAME}-*
}
trap clean_up EXIT

# Requires linuxkit/kernel-zfs:test-local to already exist in the local
# docker image store (build it from kernel/common/build-zfs.yml). Not
# pulled from any registry, so this test is skipped by default.
docker build -t zfs-check-test .
linuxkit build --docker --format kernel+initrd --name "${NAME}" test.yml
RESULT="$(linuxkitrun ${NAME})"
echo "${RESULT}"
echo "${RESULT}" | grep -q "suite PASSED"

exit 0
