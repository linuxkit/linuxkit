#!/bin/sh

set -e

# arguments are image name, prefix, then arguments passed to Docker
# eg ./riddler.sh alpine:3.4 / --read-only alpine:3.4 ls
# This script will output a tarball under prefix/ with rootfs and config.json

IMAGE="$1"; shift
PREFIX="$1"; shift

cd /tmp
mkdir -p /tmp/$PREFIX
cd /tmp/$PREFIX

# riddler always adds the apparmor options if this is not present
EXTRA_OPTIONS="--security-opt apparmor=unconfined"

ARGS="$@"
CONTAINER=$(docker create $EXTRA_OPTIONS $ARGS)
riddler $CONTAINER > /dev/null
docker rm $CONTAINER > /dev/null

# unfixed known issues
# noNewPrivileges is always set by riddler, but that is fine for our use cases

# These fixes should be removed when riddler is fixed
# process.rlimits, just a constant at present, not useful
# memory swappiness is too big by default
# remove user namespaces
# --read-only sets /dev ro
# /sysfs ro unless privileged - cannot detect so will do if grant all caps
# ipc, uts namespaces always isolated

UTS="."
IPC="."
echo $ARGS | grep -q uts=host && UTS=".linux.namespaces = (.linux.namespaces|map(select(.type!=\"uts\")))"
echo $ARGS | grep -q ipc=host && IPC=".linux.namespaces = (.linux.namespaces|map(select(.type!=\"ipc\")))"

mv config.json config.json.orig
cat config.json.orig | \
  jq "$UTS" | \
  jq "$IPC" | \
  jq 'del(.process.rlimits)' | \
  jq 'del (.linux.resources.memory.swappiness)' | \
  jq 'del(.linux.uidMappings) | del(.linux.gidMappings) | .linux.namespaces = (.linux.namespaces|map(select(.type!="user")))' | \
  jq 'if .root.readonly==true then .mounts = (.mounts|map(if .destination=="/dev" then .options |= .+ ["ro"] else . end)) else . end' | \
  jq '.mounts = if .process.capabilities | length != 38 then (.mounts|map(if .destination=="/sys" then .options |= .+ ["ro"] else . end)) else . end' \
  > config.json
rm config.json.orig

# extract rootfs
EXCLUDE="--exclude .dockerenv --exclude Dockerfile \
        --exclude dev/console --exclude dev/pts --exclude dev/shm \
        --exclude etc/hostname --exclude etc/hosts --exclude etc/resolv.conf"
mkdir -p rootfs
CONTAINER="$(docker create $IMAGE /dev/null)"
docker export "$CONTAINER" | tar -xf - -C rootfs $EXCLUDE
docker rm "$CONTAINER" > /dev/null

cd /tmp
tar cf - .
