#!/bin/sh
# SUMMARY: Check that tar output format build is reproducible
# LABELS:

set -e

# Source libraries. Uncomment if needed/defined
#. "${RT_LIB}"
. "${RT_PROJECT_ROOT}/_lib/lib.sh"

# need to build the special dir /tmp/bar12345 first
TMPDIR=/tmp/foo
TMPDIR1=${TMPDIR}/one
TMPDIR2=${TMPDIR}/two
TMPEXPORT=$(mktemp -d)
CACHE_DIR=$(mktemp -d)

clean_up() {
	rm -rf ${TMPDIR} ${CACHE_DIR} ${TMPEXPORT}
}
trap clean_up EXIT

for i in "${TMPDIR1}" "${TMPDIR2}"; do
    rm -rf "${i}"
    mkdir -p "${i}"
    echo "This is a test file for the special build arg" > "${i}/test"
    cat > "${i}/build.yml" <<EOF
org: linuxkit
image: hashes-in-build-args-$(basename "${i}")
EOF
done
git -C "${TMPDIR}" init
git -C "${TMPDIR}" config user.email "you@example.com"
git -C "${TMPDIR}" config user.name "Your Name"
git -C "${TMPDIR}" add .
git -C "${TMPDIR}" commit -m "Initial commit for special build arg test"

expected1=$(linuxkit pkg show-tag "${TMPDIR1}")
expected2=$(linuxkit pkg show-tag "${TMPDIR2}")

# print it out for the logs
echo "Building packages with special build args from ${TMPDIR1} and ${TMPDIR2}"
linuxkit --cache ${CACHE_DIR} pkg show-tag "${TMPDIR1}"
linuxkit --cache ${CACHE_DIR} pkg show-tag "${TMPDIR2}"

## Run two tests: with build args in yaml and with build arg file
targetarch="arm64"
for yml in build.yml build-file.yml; do
    extra_args=""
    if [ "$yml" = "build-file.yml" ]; then
        extra_args="--build-arg-file ./build-args"
    fi
    linuxkit --cache ${CACHE_DIR} pkg build --platforms linux/${targetarch} --force --build-yml "${yml}" . ${extra_args} 2>&1
    if [ $? -ne 0 ]; then
        echo "Build failed"
        exit 1
    fi

    current=$(linuxkit pkg show-tag --build-yml "${yml}" .)

    # for debugging
    find ${CACHE_DIR} -ls
    cat ${CACHE_DIR}/index.json
    index=$(cat ${CACHE_DIR}/index.json | jq -r '.manifests[] | select(.annotations["org.opencontainers.image.ref.name"] == "'${current}'") | .digest' | cut -d: -f2)
    echo "Current package index: ${index}"
    cat ${CACHE_DIR}/blobs/sha256/${index} | jq '.'
    manifest=$(cat ${CACHE_DIR}/blobs/sha256/${index} | jq -r '.manifests[] | select(.platform.architecture == "'${targetarch}'") | .digest' | cut -d: -f2)
    echo "Current package manifest: ${manifest}"
    cat ${CACHE_DIR}/blobs/sha256/${manifest} | jq '.'


    # dump it to a filesystem
    rm -rf "${TMPEXPORT}/*"
    linuxkit --cache ${CACHE_DIR} cache export --platform linux/${targetarch} --format filesystem --outfile /tmp/lktout123.tar "${current}"
    file /tmp/lktout123.tar
    ls -l /tmp/lktout123.tar
    cat /tmp/lktout123.tar | tar -C "${TMPEXPORT}" -xvf -
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
done

exit 0
