#! /bin/sh

# This script pushes a multiarch manifest for packages and signs it.
# 
# The TARGET must be of the form <org>/<image>:<tag> and this is what
# the manifest is pushed to. It assumes that there is are images of
# the form <org>/<image>:<tag>-<arch> already on hub.
#
# If TRUST is not set, the manifest will not be signed.

TARGET=$1
TRUST=$2

NOTARY_DELEGATION_PASSPHRASE="$DOCKER_CONTENT_TRUST_REPOSITORY_PASSPHRASE"

REPO=$(echo "$TARGET" | cut -d':' -f1)
TAG=$(echo "$TARGET" | cut -d':' -f2)

# Work out credentials. On macOS they are needed for manifest-tool and
# we need them for notary on all platforms.
case $(uname -s) in
    Darwin)
        CRED=$(echo "https://index.docker.io/v1/" | /Applications/Docker.app/Contents/Resources/bin/docker-credential-osxkeychain.bin get)
        USER=$(echo "$CRED" | jq -r '.Username')
        PASS=$(echo "$CRED" | jq -r '.Secret')
        MT_ARGS="--username $USER --password $PASS"
        ;;
    Linux)
        CRED=$(cat ~/.docker/config.json | jq -r '.auths."https://index.docker.io/v1/".auth' | base64 -d -)
        USER=$(echo $CRED | cut -d ':' -f 1)
        PASS=$(echo $CRED | cut -d ':' -f 2-)
        # manifest-tool can use docker credentials directly
        MT_ARGS=
        ;;
    *)
        echo "Unsupported platform"
        exit 1
        ;;
esac

# Push manifest list
OUT=$(manifest-tool $MT_ARGS push from-args \
                    --ignore-missing \
                    --platforms linux/amd64,linux/arm64 \
                    --template "$TARGET"-ARCH \
                    --target "$TARGET")

echo "$OUT"
if [ -z ${TRUST+x} ]; then
    echo "Not signing $TARGET"
    exit 0
fi

# Extract sha256 and length from the manifest-tool output
SHA256=$(echo "$OUT" | cut -d' ' -f2 | cut -d':' -f2)
LEN=$(echo "$OUT" | cut -d' ' -f3)

# Sign manifest (TODO: Use $USER and $PASS and pass them into notary)
notary -s https://notary.docker.io \
       -d ~/.docker/trust addhash \
       -p docker.io/"$REPO" \
       "$TAG" "$LEN" --sha256 "$SHA256" \
       -r targets/releases

echo "New multi-arch image: $REPO:$TAG"
