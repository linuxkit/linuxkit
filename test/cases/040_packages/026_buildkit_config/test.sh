#!/bin/sh
# SUMMARY: Check that we can access a registry with auth
# LABELS:

set -e

# Source libraries. Uncomment if needed/defined
#. "${RT_LIB}"
. "${RT_PROJECT_ROOT}/_lib/lib.sh"

clean_up() {
	[ -n "${CACHEDIR}" ] && rm -rf "${CACHEDIR}"
}
trap clean_up EXIT

# determine platform
ARCH=$(uname -m)
if [ "${ARCH}" = "x86_64" ]; then
  ARCH="amd64"
elif [ "${ARCH}" = "aarch64" ]; then
  ARCH="arm64"
fi
PLATFORM="linux/${ARCH}"

CACHEDIR=$(mktemp -d)

# tests:
# 1. build the local package with the custom buildkitd.toml - should succeed
# 2. rebuild the local package with the same buildkitd.toml - should succeed without starting a new builder container
# 3. rebuild the local package with the different buildkitd-2.toml - should succeed after starting a new builder container
if ! linuxkit --verbose 3 --cache "${CACHEDIR}" pkg build --platforms "${PLATFORM}" \
  --builder-config "$(pwd)/buildkitd.toml" --force \
  .; then 
  echo "Build 1 failed"
  exit 1
fi
CID1=$(docker inspect linuxkit-builder --format '{{.ID}}')

# get the containerd

if ! linuxkit --verbose 3 --cache "${CACHEDIR}" pkg build --platforms "${PLATFORM}" \
  --builder-config "$(pwd)/buildkitd.toml" --force \
  .; then 
  echo "Build 2 failed"
  exit 1
fi
CID2=$(docker inspect linuxkit-builder --format '{{.ID}}')

if ! linuxkit --verbose 3 --cache "${CACHEDIR}" pkg build --platforms "${PLATFORM}" \
  --builder-config "$(pwd)/buildkitd-2.toml" --force \
  .; then 
  echo "Build 3 failed"
  exit 1
fi
CID3=$(docker inspect linuxkit-builder --format '{{.ID}}')

# CID1 and CID2 should match, CID3 should not
echo "CID1: ${CID1}"
echo "CID2: ${CID2}"
echo "CID3: ${CID3}"

if [ "${CID1}" = "${CID2}" ] && [ "${CID2}" != "${CID3}" ]; then
  echo "Build 1 and 2 used the same builder container, but Build 3 used a different one"
else
  echo "Unexpected builder container behavior"
  exit 1
fi

exit 0
