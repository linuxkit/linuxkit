#!/bin/sh
# SUMMARY: Check that tar output format build is reproducible
# LABELS:

set -e

# Source libraries. Uncomment if needed/defined
#. "${RT_LIB}"
. "${RT_PROJECT_ROOT}/_lib/lib.sh"

# need to build the special dir /tmp/bar12345 first
TMPDIR1=/tmp/bar12345
TMPDIR2=./foo/
TMPEXPORT=$(mktemp -d)
CACHE_DIR=$(mktemp -d)

clean_up() {
	rm -rf ${TMPDIR1} ${TMPDIR2} ${CACHE_DIR} ${TMPEXPORT}
}
trap clean_up EXIT

for i in "${TMPDIR1}" "${TMPDIR2}"; do
    rm -rf "${i}"
    mkdir -p "${i}"
    echo "This is a test file for the special build arg" > "${i}/test"
    cat > "${i}/build.yml" <<EOF
org: linuxkit
image: hashes-in-build-args
EOF
    git -C "${i}" init
    git -C "${i}" config user.email "you@example.com"
    git -C "${i}" config user.name "Your Name"
    git -C "${i}" add .
    git -C "${i}" commit -m "Initial commit for special build arg test"
done

# print it out for the logs
echo "Building packages with special build args from ${TMPDIR1} and ${TMPDIR2}"
linuxkit --cache ${CACHE_DIR} pkg show-tag "${TMPDIR1}"
linuxkit --cache ${CACHE_DIR} pkg show-tag "${TMPDIR2}"

logs=$(linuxkit --cache ${CACHE_DIR} pkg build --force . 2>&1)
if [ $? -ne 0 ]; then
    echo "Build failed with logs:"
    echo "${logs}"
    exit 1
fi

expected1=$(linuxkit pkg show-tag "${TMPDIR1}")
expected2=$(linuxkit pkg show-tag "${TMPDIR2}")
current=$(linuxkit pkg show-tag .)

# dump it to a filesystem
linuxkit --cache ${CACHE_DIR} cache export --format filesystem --outfile - "${current}" | tar -C "${TMPEXPORT}" -xvf -
# for extra debugging
actual1=$(cat ${TMPEXPORT}/var/hash1)
actual2=$(cat ${TMPEXPORT}/var/hash2)


if [ "${expected1}" != "${actual1}" ]; then
    echo "Expected HASH1: ${expected1}, but got: ${actual1}"
    exit 1
fi

if [ "${expected2}" != "${actual2}" ]; then
    echo "Expected HASH2: ${expected2}, but got: ${actual2}"
    exit 1
fi

# Check that the build args were correctly transformed

exit 0
