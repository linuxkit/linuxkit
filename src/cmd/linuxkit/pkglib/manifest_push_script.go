package pkglib

const manifestPushScript = `
#! /bin/sh

set -e

# This script pushes a multiarch manifest for packages and signs it.
#
# The TARGET must be of the form <org>/<image>:<tag> and this is what
# the manifest is pushed to. It assumes that there is are images of
# the form <org>/<image>:<tag>-<arch> already on hub.
#
# If TRUST is not set, the manifest will not be signed.
#
# For signing, DOCKER_CONTENT_TRUST_REPOSITORY_PASSPHRASE must be set.

# This should all be replaced with 'docker manifest' once it lands.

TARGET=$1
TRUST=$2

REPO=$(echo "$TARGET" | cut -d':' -f1)
TAG=$(echo "$TARGET" | cut -d':' -f2)

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
OUT=$(manifest-tool $MT_ARGS push from-args \
                    --ignore-missing \
                    --platforms linux/amd64,linux/arm64,linux/s390x \
                    --template "$TARGET"-ARCH \
                    --target "$TARGET")

echo "$OUT"
if [ -z "$TRUST" ]; then
    echo "Not signing $TARGET"
    exit 0
fi

# Extract sha256 and length from the manifest-tool output
SHA256=$(echo "$OUT" | cut -d' ' -f2 | cut -d':' -f2)
LEN=$(echo "$OUT" | cut -d' ' -f3)

# notary 0.6.0 accepts authentication as base64-encoded "username:password"
export NOTARY_AUTH=$(echo "$USER:$PASS" | base64)
export NOTARY_DELEGATION_PASSPHRASE="$DOCKER_CONTENT_TRUST_REPOSITORY_PASSPHRASE"

notary -s https://notary.docker.io -d $HOME/.docker/trust addhash \
       -p docker.io/$REPO $TAG $LEN --sha256 $SHA256 \
       -r targets/releases

echo
echo "New signed multi-arch image: $REPO:$TAG"
echo
`
