#!/bin/sh

# FOR REFERENCE ONLY
# needs adjusting for real use, riddler needs some updates

set -e

printf "FROM scratch\nCOPY . ./\n" > rootfs/Dockerfile
IMAGE=$(docker build -q rootfs)
CONTAINER=$(docker create --net=none --security-opt apparmor=unconfined --cap-drop all --read-only -v /proc/sys/fs/binfmt_misc:/binfmt_misc $IMAGE /usr/bin/binfmt -dir /etc/binfmt.d/ -mount /binfmt_misc)
rm rootfs/Dockerfile
docker run -v $PWD:/conf -v /var/run/docker.sock:/var/run/docker.sock --rm jess/riddler -f -bundle /conf $CONTAINER
docker rm $CONTAINER
