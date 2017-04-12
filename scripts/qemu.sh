#!/bin/sh

QEMU_IMAGE=linuxkit/qemu:4563d58e97958f4941fbef9e74cabc08bd402144@sha256:b2db0b13ba1cbb6b48218f088fe0a4d860e1db2c4c6381b5416536f48a612230

# if not interactive
if [ ! -t 0 -a -z "$1" ]
then
	# non interactive, tarball input
	docker run -i --rm "$QEMU_IMAGE"
	exit $?
fi

FILE=$1
FILE2=$2
CMDLINE=$3
[ -z "$FILE" ] && FILE="$PWD/moby"

BASE=$(basename "$FILE")
DIR=$(dirname "$FILE")
if [ ! -f "$FILE" -a -f $DIR/$BASE-initrd.img -a -f $DIR/$BASE-bzImage ]
then
	FILE=$DIR/$BASE-initrd.img
	FILE2=$DIR/$BASE-bzImage
fi

echo "$FILE" | grep -q '^/' || FILE="$PWD/$FILE"
if [ ! -z "$FILE2" ]
then
	echo "$FILE2" | grep -q '^/' || FILE2="$PWD/$FILE2"
fi
if [ ! -z "$CMDLINE" ]
then
	echo "$CMDLINE" | grep -q '^/' || CMDLINE="$PWD/$CMDLINE"
fi

if [ -c "/dev/kvm" ] ; then
    DEVKVM="--device=/dev/kvm"
fi
BASE=$(basename "$FILE")
MOUNTS="-v $FILE:/tmp/$BASE"
BASE2=$(basename "$FILE2")
BASE3=$(basename "$CMDLINE")

[ ! -z "$FILE2" ] && MOUNTS="$MOUNTS -v $FILE2:/tmp/$BASE2"
[ ! -z "$CMDLINE" ] && MOUNTS="$MOUNTS -v $CMDLINE:/tmp/$BASE3"

docker run -it --rm $MOUNTS $DEVKVM "$QEMU_IMAGE"
