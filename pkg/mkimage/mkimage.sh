#!/bin/sh

# currently only supports ext4 disks; will extend for other filesystems, ISO, ...

do_mkfs()
{
	diskdev="$1"

	# new disks does not have an DOS signature in sector 0
	# this makes sfdisk complain. We can workaround this by letting
	# fdisk create that DOS signature, by just do a "w", a write.
	# http://bugs.alpinelinux.org/issues/145
	echo "w" | fdisk $diskdev >/dev/null

	# format one large partition
	echo ";" | sfdisk --quiet $diskdev

	# update status
	blockdev --rereadpt $diskdev 2> /dev/null

	# wait for device
	for i in $(seq 1 50); do test -b "$DATA" && break || sleep .1; mdev -s; done

	FSOPTS="-O resize_inode,has_journal,extent,huge_file,flex_bg,uninit_bg,64bit,dir_nlink,extra_isize"

	mkfs.ext4 -q -F $FSOPTS ${diskdev}1
}

DEV="$(find /dev -maxdepth 1 -type b ! -name 'loop*' | grep -v '[0-9]$' | sed 's@.*/dev/@@' | sort | head -1 )"
DEV2="$(find /dev -maxdepth 1 -type b ! -name 'loop*' | grep -v '[0-9]$' | sed 's@.*/dev/@@' | sort | head -2 | tail -1)"

[ -z "${DEV}" ] && exit 1
[ -z "${DEV2}" ] && exit 1

DRIVE="/dev/${DEV}"
DRIVE2="/dev/${DEV2}"

# format
do_mkfs "$DRIVE"

PARTITION="${DRIVE}1"

# mount
mount "$PARTITION" /mnt

# copy kernel, initrd from tarball on second disk
tar xf "${DRIVE2}" -C /mnt

# rename if they do not have canonical names
(
	cd /mnt
	[ -f kernel ] || mv *-kernel kernel
	[ -f initrd.img ] || mv *-initrd.img initrd.img
	[ -f cmdline ] || mv *-cmdline cmdline
)

# create syslinux.cfg
CMDLINE="$(cat /mnt/cmdline)"
rm -f /mnt/cmdline

CFG="DEFAULT linux
LABEL linux
    KERNEL /kernel
    INITRD /initrd.img
    APPEND ${CMDLINE}
"

printf "$CFG" > /mnt/syslinux.cfg

# install syslinux
extlinux --install /mnt

# unmount
umount /mnt

# install mbr
dd if=usr/share/syslinux/mbr.bin of="$DRIVE" bs=440 count=1

# make bootable
sfdisk -A "$DRIVE" 1
