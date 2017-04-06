#!/bin/sh

QEMU_IMAGE=mobylinux/qemu:e8d64debb343de87d66aafb21502808409600a04@sha256:e84e13549a9b5f6959bea258356995722ed5c8c477a9064ef747c32bc5a97917

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
