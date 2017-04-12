#!/bin/sh

set -e

# arguments are image name
# This script will output a tarball, suitable to be turned into a cpio archive
# This is a bit hacky, should be improved later, as it hardcodes config.

IMAGE="$1"; shift

cd /tmp

# extract rootfs
EXCLUDE="--exclude .dockerenv --exclude Dockerfile \
        --exclude dev/console --exclude dev/pts --exclude dev/shm \
        --exclude etc/hostname --exclude etc/hosts --exclude etc/mtab --exclude etc/resolv.conf"

CONTAINER="$(docker create $IMAGE /dev/null)"
docker export "$CONTAINER" | tar -xf - $EXCLUDE
docker rm "$CONTAINER" > /dev/null

# these three files are bind mounted in by docker so they are not what we want

mkdir -p etc

cat << EOF > etc/hosts
127.0.0.1	localhost
::1	localhost ip6-localhost ip6-loopback
fe00::0	ip6-localnet
ff00::0	ip6-mcastprefix
ff02::1	ip6-allnodes
ff02::2	ip6-allrouters
EOF

cat << EOF > etc/resolv.conf
nameserver 8.8.8.8
nameserver 8.8.4.4
nameserver 2001:4860:4860::8888
nameserver 2001:4860:4860::8844
EOF

printf 'linuxkit' > etc/hostname

ln -s /proc/mounts etc/mtab

tar cf - .
