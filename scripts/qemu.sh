#!/bin/sh

QEMU_IMAGE=mobylinux/qemu:ac2f8d3258d09541f0dab7529f62bb7e9b9bb79c@sha256:b60d1421937b0ebf4846341a1cda1ca00f44a45553d5494080b2ca8aac468773

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
