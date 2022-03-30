#!/bin/sh

# common tags tests

#!/bin/sh
# SUMMARY: Test existence and correctness of kernel builder tag, label and file
# LABELS:
# REPEAT:

set -e

# Source libraries. Uncomment if needed/defined
#. "${RT_LIB}"
. "${RT_PROJECT_ROOT}/_lib/lib.sh"

if [ -z "${KERNEL}" ]; then
    echo "KERNEL env var must be set"
    exit 1
fi

NAME=tags

clean_up() {
    docker rm ${ctrid}
    /bin/rm -f ${BUILDERFILE}
}
trap clean_up EXIT

# check the kernel images for tags, labels, files
BUILDER=${KERNEL}-builder
BUILDERFILE=/tmp/kernel-builder-$$

docker pull ${KERNEL}
docker pull ${BUILDER}
BUILDERLABEL=$(docker inspect -f '{{index .Config.Labels "org.mobyproject.linuxkit.kernel.buildimage"}}'  ${KERNEL})
# create the container; /bin/sh does not exist, but that does not prevent us from indicating what the command
#   *would* be. Indeed, you *must* have a command for `docker create` to work
ctrid=$(docker create $KERNEL /bin/sh)
docker cp ${ctrid}:/kernel-builder ${BUILDERFILE}
FILECONTENTS=$(cat ${BUILDERFILE})

# Get a list of architectures for which we have this kernel
KERNEL_ARCHES=$(docker manifest inspect ${KERNEL} | jq  -r -c '.manifests[].platform.architecture')

# Get builder manifest
BUILDER_MANIFEST=$(docker manifest inspect ${BUILDER} | jq  -c '.manifests')

# Get the manifest of the builder pointed to by the label
BUILDER_LABEL_MANIFEST=$(docker manifest inspect ${BUILDERLABEL} | jq  -c '.manifests')


# these two files should be identical
echo "builder label: ${BUILDERLABEL}"
echo "builder file: ${FILECONTENTS}"
echo "builder tag: ${BUILDER}"

# check that the label and file contents match
if [ "${BUILDERLABEL}" != "${FILECONTENTS}" ]; then
    echo "label vs file contents mismatch"
    exit 1
fi

# Check that for each architecture we have the kernel for builder and the builder label points to the same thing
for ARCH in ${KERNEL_ARCHES}; do
    BUILDER_ARCH_DIGEST=$(echo ${BUILDER_MANIFEST} | jq -r --arg ARCH "$ARCH" '.[] | select (.platform.architecture == $ARCH) | .digest')
    BUILDER_LABEL_ARCH_DIGEST=$(echo ${BUILDER_LABEL_MANIFEST} | jq -r --arg ARCH "$ARCH" '.[] | select (.platform.architecture == $ARCH) | .digest')

    if [ -z "${BUILDER_ARCH_DIGEST}" ]; then
        echo "No Builder for ${ARCH} in manifest ${BUILDER}"
        exit 1
    fi
    if [ -z "${BUILDER_LABEL_ARCH_DIGEST}" ]; then
        echo "No Builder for ${ARCH} in manifest ${BUILDERLABEL}"
        exit 1
    fi

    if [ "${BUILDER_ARCH_DIGEST}" != "${BUILDER_LABEL_ARCH_DIGEST}" ]; then
        echo "Builder digests for kernel ${KERNEL} arch ${ARCH} do not match ${BUILDER_ARCH_DIGEST} != ${BUILDER_LABEL_ARCH_DIGEST}"
        exit 1
    fi

    echo "Builder tags/labels for kernel ${KERNEL} arch ${ARCH} match: ${BUILDER_ARCH_DIGEST} == ${BUILDER_LABEL_ARCH_DIGEST}"
done

exit 0
