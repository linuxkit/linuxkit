#!/bin/sh
# SUMMARY: Check that we go through the mirror when building, and fail if mirror configured but not provided
# LABELS:

set -e

# Source libraries. Uncomment if needed/defined
#. "${RT_LIB}"
. "${RT_PROJECT_ROOT}/_lib/lib.sh"

clean_up() {
	docker kill "${REGISTRY_NAME}" || true
	[ -n "${CACHEDIR}" ] && rm -rf "${CACHEDIR}"
	[ -n "${REGISTRY_DIR}" ] && rm -rf "${REGISTRY_DIR}"
}
trap clean_up EXIT

# container names
REGISTRY_NAME="test-registry-$$"
REGISTRY_DIR=$(mktemp -d)
CACHEDIR=$(mktemp -d)



# 2 tests:
# 1. build a package configured to use a mirror without starting mirror - should fail
# 2. build a package configured to use a mirror after starting mirror - should succeed
if linuxkit --mirror http://localhost:5001 --cache ${CACHEDIR} build --format kernel+initrd --name "${NAME}" ./test.yml; then
  echo "Test 1 failed: build succeeded without starting mirror"
  exit 1
fi

# Start registry
REGISTRY_CID=$(docker run -d --rm -v $(pwd)/config.yml:/etc/distribution/config.yml --name ${REGISTRY_NAME} -p 5001:5000 registry:3)

# this one should succeed
linuxkit --mirror http://localhost:5001 --cache ${CACHEDIR} build --format kernel+initrd --name "${NAME}" ./test.yml

exit 0
