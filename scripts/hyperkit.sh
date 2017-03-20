#!/bin/sh

set -e

DOCKER_HYPERKIT=/Applications/Docker.app/Contents/MacOS/com.docker.hyperkit
DOCKER_VPNKIT=/Applications/Docker.app/Contents/MacOS/vpnkit

[ -f bin/com.docker.hyperkit ] && HYPERKIT=bin/com.docker.hyperkit
[ -f bin/vpnkit ] && VPNKIT=bin/vpnkit

[ -f "$DOCKER_HYPERKIT" ] && HYPERKIT="$DOCKER_HYPERKIT"
[ -f "$DOCKER_VPNKIT" ] && VPNKIT="$DOCKER_VPNKIT"

command -v com.docker.hyperkit > /dev/null && HYPERKIT="$(command -v com.docker.hyperkit)"
command -v hyperkit > /dev/null && HYPERKIT="$(command -v hyperkit)"
command -v vpnkit > /dev/null && VPNKIT="$(command -v vpnkit)"

if [ $# -eq 0 ]
then
	PREFIX="moby"
	KERNEL="$PREFIX-bzImage"
	INITRD="$PREFIX-initrd.img"
	CMDLINE="$PREFIX-cmdline"
elif [ $# -eq 1 ]
then
	PREFIX="$1"
	KERNEL="$PREFIX-bzImage"
	INITRD="$PREFIX-initrd.img"
	CMDLINE="$PREFIX-cmdline"
else
	KERNEL=$1
	INITRD=$2
	CMDLINE=$3
fi

# TODO start vpnkit if Docker for Mac not running
SLIRP_SOCK="$HOME/Library/Containers/com.docker.docker/Data/s50"

[ -f disk.img ] || dd if=/dev/zero of=disk.img bs=1048576 count=256

MEM="-m 1G"
SMP="-c 1"
NET="-s 2:0,virtio-vpnkit,path=$SLIRP_SOCK"
IMG_HDD="-s 4,virtio-blk,disk.img"
PCI_DEV="-s 0:0,hostbridge -s 31,lpc"
RND="-s 5,virtio-rnd"
LPC_DEV="-l com1,stdio"

#$VPNKIT --ethernet $SLIRP_SOCK &>/dev/null &
#trap "kill $!; rm $SLIRP_SOCK" EXIT

$HYPERKIT -A $MEM $SMP $PCI_DEV $LPC_DEV $NET $IMG_HDD $RND -u -f kexec,$KERNEL,$INITRD,"$CMDLINE"
