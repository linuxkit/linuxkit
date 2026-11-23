#! /bin/sh
set -e

# This script creates a multiarch manifest for the 'linuxkit/alpine'
# image and pushes it. The manifest is pushed with the tag of
# the amd64 images (which is the suffix removed). On macOS we use the
# credentials helper to extract the Hub credentials.
#
# This script is specific to 'linuxkit/alpine'. For normal packages we
# use a different scheme.
#
# This should all be replaced with 'docker manifest' once it lands.

ORG=$1
IMAGE=$2

IMG_X86_64=$(head -1 versions.x86_64 | sed 's,[#| ]*,,')
IMG_ARM64=$(head -1 versions.aarch64 | sed 's,[#| ]*,,')
IMG_RISCV64=$(head -1 versions.riscv64 | sed 's,[#| ]*,,')
# Extract the TAG from the tree hash - just like how "linuxkit pkg show-tag" does it - name and build the manifest target name
TAG=$(git ls-tree --full-tree HEAD -- $(pwd) | awk '{print $3}')
DIRTY=$(git diff-index HEAD -- $(pwd))
if [ -n "$DIRTY" ]; then
  echo "will not push out manifest when git tree is dirty" >&2
  exit 1
fi
TARGET="$ORG/$IMAGE:$TAG"

docker buildx imagetools create --tag "$TARGET" \
    "$IMG_ARM64" \
    "$IMG_X86_64" \
    "$IMG_RISCV64"

