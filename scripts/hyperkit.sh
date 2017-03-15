#!/bin/sh

set -e

if [ $# -eq 0 ]
then
	PREFIX="moby"
	KERNEL="$PREFIX-bzImage"
	INITRD="$PREFIX-initrd.img"
	CMDLINE=$(bin/moby --cmdline ${PREFIX}.yaml)
elif [ $# -eq 1 ]
then
	PREFIX="$1"
	KERNEL="$PREFIX-bzImage"
	INITRD="$PREFIX-initrd.img"
	CMDLINE=$(bin/moby --cmdline ${PREFIX}.yaml)
else
	KERNEL=$1
	INITRD=$2
	CMDLINE=$3
fi

SLIRP_SOCK="$HOME/Library/Containers/com.docker.docker/Data/s50"

[ -f disk.img ] || dd if=/dev/zero of=disk.img bs=1048576 count=256

MEM="-m 1G"
SMP="-c 1"
NET="-s 2:0,virtio-vpnkit,path=$SLIRP_SOCK"
IMG_HDD="-s 4,virtio-blk,disk.img"
PCI_DEV="-s 0:0,hostbridge -s 31,lpc"
RND="-s 5,virtio-rnd"
LPC_DEV="-l com1,stdio"

#bin/vpnkit --ethernet $SLIRP_SOCK &>/dev/null &
#trap "kill $!; rm $SLIRP_SOCK" EXIT

bin/com.docker.hyperkit -A $MEM $SMP $PCI_DEV $LPC_DEV $NET $IMG_HDD $RND -u -f kexec,$KERNEL,$INITRD,"$CMDLINE"
