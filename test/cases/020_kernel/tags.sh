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

# get the manifests for the referenced tag and for the referenced builder.
# these are not guaranated to be identical, since the orders can change. So we need to account for that.
sumtag=$(docker manifest inspect ${BUILDER} | jq  -c '.manifests | sort_by(.digest)' | sha256sum | awk '{print $1}')
sumlabel=$(docker manifest inspect ${BUILDERLABEL} | jq  -c '.manifests | sort_by(.digest)' | sha256sum | awk '{print $1}')

# these two files should be identical
echo "builder label: ${BUILDERLABEL}"
echo "builder file: ${FILECONTENTS}"
echo "builder tag: ${BUILDER}"
echo "builder tag sha256: ${sumtag}"
echo "builder label sha256: ${sumlabel}"

# check that the label and file contents match
if [ "${BUILDERLABEL}" != "${FILECONTENTS}" ]; then
	echo "label vs file contents mismatch"
	exit 1
fi
# check that the tag actually points to the manifest
if [ "${sumtag}" != "${sumlabel}" ]; then
	echo "tag ${BUILDER} and label ${BUILDERLABEL} have mismatched contents"
	exit 1
fi


exit 0
