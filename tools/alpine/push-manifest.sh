#! /bin/sh

# This script creates a multiarch manifest for the 'linuxkit/alpine'
# image, pushes and signs it. The manifest is pushed with the tag of
# the amd64 images (which is the suffix removed). On macOS we use the
# credentials helper to extract the Hub credentials. We need to
# manually sign the manifest using 'notary'.
#
# This script is specific to 'linuxkit/alpine'. For normal packages we
# use a different scheme.
#
# This should all be replaced with 'docker manifest' once it lands.

ORG=$1
IMAGE=$2

IMG_X86_64=$(head -1 versions.x86_64 | sed 's,[#| ]*,,')
IMG_ARM64=$(head -1 versions.aarch64 | sed 's,[#| ]*,,')
IMG_MANIFEST=$(echo "$IMG_X86_64" | sed 's,\-.*$,,')
IMG_TAG=$(echo "$IMG_MANIFEST" | sed 's,.*:,,')

YAML=$(mktemp)
cat <<EOF > "$YAML"
image: $IMG_MANIFEST
manifests:
  - image: $IMG_ARM64
    platform:
      architecture: arm64
      os: linux
  - image: $IMG_X86_64
    platform:
      architecture: amd64
      os: linux
EOF

# work out additional arguments. Specifically, on Darwin the hub
# credentials are stored on the keychain and we need to extract them
# from there
case $(uname -s) in
    Darwin)
        CRED=$(echo "https://index.docker.io/v1/" | /Applications/Docker.app/Contents/Resources/bin/docker-credential-osxkeychain.bin get)
        USER=$(echo "$CRED" | jq -r '.Username')
        PASS=$(echo "$CRED" | jq -r '.Secret')
        USERPASS="$USER\n$PASS"
        MT_ARGS="--username $USER --password $PASS"
        ;;
    Linux)
        MT_ARGS=
        USERPASS=$(cat ~/.docker/config.json | jq -r '.auths."https://index.docker.io/v1/".auth' | base64 -d - | sed 's,:,\\n,')
        ;;
    *)
        echo "Unsupported platform"
        exit 1
        ;;
esac

# Push manifest list
OUT=$(manifest-tool $MT_ARGS push from-spec "$YAML")
rm "$YAML"
echo "$OUT"
SHA256=$(echo "$OUT" | cut -d' ' -f2 | cut -d':' -f2)
LEN=$(echo "$OUT" | cut -d' ' -f3)

# Sign manifest (TODO: Use $USERPASS and pass them into notary)
notary -s https://notary.docker.io \
       -d ~/.docker/trust addhash \
       -p docker.io/"$ORG"/"$IMAGE" \
       "$IMG_TAG" "$LEN" --sha256 "$SHA256" \
       -r targets/releases

echo "New multi-arch image: $ORG/$IMAGE:$IMG_TAG"
