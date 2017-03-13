#!/bin/sh

QEMU_IMAGE=mobylinux/qemu:97973fb6721778c639676812ccb8bc3332e0a542@sha256:c08dac641a75fda3232a8ff3250f23d743aeac12aa4db02ec7926a42b79b0e69

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

if [ -c "/dev/kvm" ] ; then
    DEVKVM="--device=/dev/kvm"
fi
BASE=$(basename "$FILE")
MOUNTS="-v $FILE:/tmp/$BASE"
BASE2=$(basename "$FILE2")
[ ! -z "$FILE2" ] && MOUNTS="$MOUNTS -v $FILE2:/tmp/$BASE2"
docker run -it --rm $MOUNTS $DEVKVM "$QEMU_IMAGE" $CMDLINE
