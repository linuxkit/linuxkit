#!/bin/sh

QEMU_IMAGE=mobylinux/qemu:156d2160c2ccf4d5118221bc2708f6c0981d54cc@sha256:e1345ba0400d6c45bf3bdf4f4ed425c3d7596d11e6553b83f17f5893dfc49f7b

# if not interactive
if [ ! -t 0 -a -z "$1" ]
then
	# non interactive, tarball input
	docker run -i --rm "$QEMU_IMAGE"
	exit $?
fi

FILE=$1
FILE2=$2
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

BASE=$(basename "$FILE")
MOUNTS="-v $FILE:/tmp/$BASE"
BASE2=$(basename "$FILE2")
[ ! -z "$FILE2" ] && MOUNTS="$MOUNTS -v $FILE2:/tmp/$BASE2"
docker run -it --rm $MOUNTS "$QEMU_IMAGE"
