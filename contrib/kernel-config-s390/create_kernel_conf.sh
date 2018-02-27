#!/bin/sh

if [ -z "$GOPATH" ]; then
	>&2 echo "You need to set the GOPATH"
	exit 1
fi

DIR=$GOPATH/src/github.com/linuxkit/linuxkit
KERNEL_DIR=$DIR/kernel
CONTRIB=$DIR/contrib/kernel-config-s390
DOCKER_IMG=linuxkit/kconfig
set -x
set -e

cd $KERNEL_DIR

for v in sources/*.tar.xz
do
    KERNEL_VER=$(echo "$v" |sed -e 's/^\(sources\/linux-\)//' | sed -e 's/\.[0-9]*\.tar\.xz$//')
    # Trying to stick as close as possible to x86 and arm configuration
    [ -e config-$KERNEL_VER.x-aarch64 ] || continue
    $DIR/scripts/kconfig-split.py config-$KERNEL_VER.x-aarch64 config-$KERNEL_VER.x-x86_64
    mv split-common split-common-$KERNEL_VER.x
done
docker run -ti --rm  -v $KERNEL_DIR:/src -v $CONTRIB:/entry $DOCKER_IMG -c /entry/entrypoint.sh
