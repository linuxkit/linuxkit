#!/bin/sh

set -e

KERNEL="alpine/kernel/x86_64/vmlinuz64"
INITRD="alpine/initrd.img"
CMDLINE="earlyprintk=serial console=ttyS0"

[ -f disk.img ] || dd if=/dev/zero of=disk.img bs=1m count=256

MEM="-m 1G"
SMP="-c 1"
NET=""
if (( $EUID != 0 )); then
    printf "Need to run as root to get networking!\n\n"
    exit 1
fi
NET="-s 2:0,virtio-net"
IMG_HDD="-s 4,virtio-blk,disk.img"
PCI_DEV="-s 0:0,hostbridge -s 31,lpc"
RND="-s 5,virtio-rnd"
LPC_DEV="-l com1,stdio"

hyperkit.git/build/com.docker.hyperkit -A $MEM $SMP $PCI_DEV $LPC_DEV $NET $IMG_HDD $RND -u -f kexec,$KERNEL,$INITRD,"$CMDLINE"
