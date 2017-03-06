#!/bin/sh

# This script runs a HyperKit VM reusing the Docker for Mac VPNKit.
# It requires a recent version of Docker for Mac with an updated VPNKit

set -e

HYPERKIT=/Applications/Docker.app/Contents/MacOS/com.docker.hyperkit
DB=${HOME}/Library/Containers/com.docker.docker/Data/database
DB_BRIDGE=com.docker.driver.amd64-linux/slirp/bridge-connections

# Check if VPNKit L2 bridge mode is enabled
enable_bridge() {
    (cd "${DB}" && \
         echo -n "1" > ${DB_BRIDGE} && \
         git add ${DB_BRIDGE} && \
         git commit -m "enable bridge"
    )
}

(cd "${DB}"; git reset --hard)
if [ -f "${DB}/${DB_BRIDGE}" ]; then
    content=$(cat "${DB}/${DB_BRIDGE}")
    [ "${content}" != "1" ] && enable_bridge
else
    enable_bridge
fi

KERNEL="kernel/x86_64/vmlinuz64"
: ${INITRD:="alpine/initrd.img"}
CMDLINE="earlyprintk=serial console=ttyS0"

DISK=$(mktemp disk.img.XXXX)
dd if=/dev/zero of="$DISK" bs=1048576 count=256

MEM="-m 1G"
SMP="-c 1"
SLIRP_SOCK=${HOME}/Library/Containers/com.docker.docker/Data/s50
NET="-s 2:0,virtio-vpnkit,uuid=35e617a8-5db9-4420-9dfb-84da72dec7ac,path=$SLIRP_SOCK"
IMG_HDD="-s 4,virtio-blk,$DISK"
PCI_DEV="-s 0:0,hostbridge -s 31,lpc"
RND="-s 5,virtio-rnd"
LPC_DEV="-l com1,stdio"
$HYPERKIT -A $MEM $SMP $PCI_DEV $LPC_DEV $NET $IMG_HDD $RND -u -f kexec,$KERNEL,$INITRD,"$CMDLINE"
