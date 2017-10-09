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
if [ -z "$TRUST" ]; then
    echo "Not signing $TARGET"
    exit 0
fi

# Extract sha256 and length from the manifest-tool output
SHA256=$(echo "$OUT" | cut -d' ' -f2 | cut -d':' -f2)
LEN=$(echo "$OUT" | cut -d' ' -f3)

# Notary requires a PTY for username/password so use expect for that.
export NOTARY_DELEGATION_PASSPHRASE="$DOCKER_CONTENT_TRUST_REPOSITORY_PASSPHRASE"
NOTARY_CMD="notary -s https://notary.docker.io -d $HOME/.docker/trust addhash \
             -p docker.io/$REPO $TAG $LEN --sha256 $SHA256 \
             -r targets/releases"

echo '
spawn '"$NOTARY_CMD"'
set pid [exp_pid]
set timeout 60
expect {
    timeout {
        puts "Expected username prompt"
        exec kill -9 $pid
        exit 1
    }
    "username: " {
        send "'"$USER"'\n"
    }
}
expect {
    timeout {
        puts "Expected password prompt"
        exec kill -9 $pid
        exit 1
    }
    "password: " {
        send "'"$PASS"'\n"
    }
}
expect {
    timeout {
        puts "Expected password prompt"
        exec kill -9 $pid
        exit 1
    }
    eof {
    }
}
set waitval [wait -i $spawn_id]
set exval [lindex $waitval 3]
exit $exval
' | expect -f -

echo
echo "New signed multi-arch image: $REPO:$TAG"
echo
`
