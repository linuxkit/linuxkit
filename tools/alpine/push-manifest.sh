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
IMG_s390x=$(head -1 versions.s390x | sed 's,[#| ]*,,')
# Extract the TAG from the tree hash - just like how "linuxkit pkg show-tag" does it - name and build the manifest target name
TAG=$(git ls-tree --full-tree HEAD -- $(pwd) | awk '{print $3}')
DIRTY=$(git diff-index HEAD -- $(pwd))
if [ -n "$DIRTY"]; then
  echo "will not push out manifest when git tree is dirty" >&2
  exit 1
fi
TARGET="$ORG/$IMAGE:$TAG"

YAML=$(mktemp)
cat <<EOF > "$YAML"
image: $TARGET
manifests:
  - image: $IMG_ARM64
    platform:
      architecture: arm64
      os: linux
  - image: $IMG_X86_64
    platform:
      architecture: amd64
      os: linux
  - image: $IMG_s390x
    platform:
      architecture: s390x
      os: linux
EOF

# Work out credentials. On macOS they are needed for manifest-tool and
# we need them for notary on all platforms.
case $(uname -s) in
    Darwin)
        # Prior to 2018-03-27 D4M used a .bin suffix on the keychain utility binary name. Support the old name for a while
        if [ -f /Applications/Docker.app/Contents/Resources/bin/docker-credential-osxkeychain.bin ]; then
            CREDHELPER="/Applications/Docker.app/Contents/Resources/bin/docker-credential-osxkeychain.bin"
        else
            CREDHELPER="/Applications/Docker.app/Contents/Resources/bin/docker-credential-osxkeychain"
        fi
        ;;
    Linux)
        CREDSTORE=$(cat ~/.docker/config.json | jq -r '.credsStore // empty')
        if [ -n "$CREDSTORE" ] ; then
            CREDHELPER="docker-credential-$CREDSTORE"
        else
            CRED=$(cat ~/.docker/config.json | jq -r '.auths."https://index.docker.io/v1/".auth' | base64 -d -)
            USER=$(echo $CRED | cut -d ':' -f 1)
            PASS=$(echo $CRED | cut -d ':' -f 2-)
            # manifest-tool can use docker credentials directly
            MT_ARGS=
        fi
        ;;
    *)
        echo "Unsupported platform"
        exit 1
        ;;
esac
if [ -n "$CREDHELPER" ] ; then
    CRED=$(echo "https://index.docker.io/v1/" | "$CREDHELPER" get)
    USER=$(echo "$CRED" | jq -r '.Username')
    PASS=$(echo "$CRED" | jq -r '.Secret')
    MT_ARGS="--username $USER --password $PASS"
fi

# Push manifest list
OUT=$(manifest-tool $MT_ARGS push from-spec --ignore-missing "$YAML")
rm "$YAML"
echo "$OUT"
