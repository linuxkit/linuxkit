#!/bin/sh

KERNEL="../kernel/vmlinuz64"
INITRD="../alpine/initrd.img"
CMDLINE="earlyprintk=serial console=ttyS0"

MEM="-m 1G"
#SMP="-c 2"
NET=""
if (( $EUID != 0 )); then
    printf "Warning: not running as root will have no networking!\n\n"
    sleep 1
else
NET="-s 2:0,virtio-net"
fi
#IMG_CD="-s 3,ahci-cd,/somepath/somefile.iso"
#IMG_HDD="-s 4,virtio-blk,/somepath/somefile.img"
PCI_DEV="-s 0:0,hostbridge -s 31,lpc"
RND="-s 5,virtio-rnd"
LPC_DEV="-l com1,stdio"
ACPI="-A"
CLOCK="-u"
#UUID="-U deadbeef-dead-dead-dead-deaddeafbeef"

build/xhyve $ACPI $MEM $SMP $PCI_DEV $LPC_DEV $NET $IMG_CD $IMG_HDD $RND $UUID $CLOCK -f kexec,$KERNEL,$INITRD,"$CMDLINE"
