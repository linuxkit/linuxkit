#!/bin/sh

set -e

MOBY_SRC_ROOT=${MOBY_SRC_ROOT:-/build}

arrowecho()
{
	echo " --->" "$@"
}

errecho()
{
	echo "$@" >&2
}

# $1 - the device to format (e.g. /dev/xvdb)
format_on_device()
{
	while [ ! -e "$1" ]
	do
		sleep 0.1
	done
	arrowecho "Formatting boot partition"

	# TODO (nathanleclaire): Any more readable or more elegant solution to
	# account for this minor (specify 1st partition as bootable) difference
	# between cloud builds?
	if [ "${PROVIDER}" = "aws" ]
	then
		# This heredoc might be confusing at first glance, so here is a detailed
		# summary of what each line does:
		#
		# n  - create new partition
		# p  - make it a primary partition
		# 1  - it should be partition #1
		# \n - use default first cylinder
		# \n - use default last cylinder
		# a  - toggle a partition as bootable
		# 1  - first partition
		# w  - write changes and exit
		fdisk "$1" << EOF
n
p
1


a
1
w
EOF
	elif [ ${PROVIDER} = "azure" ]
	then
		fdisk "$1" << EOF
n
p
1

+150M
a
n
p
2

+150M
n
p
3


w
EOF
	else
		errecho "Provider not recognized: ${PROVIDER}"
		exit 1
	fi

	# To ensure everything went smoothly, print the resulting partition table.
	arrowecho "Printing device partition contents"
	fdisk -l "$1"
}

# $1 - device
configure_syslinux_on_device_partition()
{
	# Mount created root partition, format it as ext4, and copy over the needed
	# files for boot (syslinux configuration, kernel binary, and initrd.img)
	while [ ! -e "$1" ]
	do
		sleep 0.1
	done

	arrowecho "Making filesystem on partition"
	mke2fs -t ext4 "$1"

	arrowecho "Mounting partition filesystem"

	ROOT_PARTITION_MOUNT="${MOBY_SRC_ROOT}/moby"
	if [ ! -d ${ROOT_PARTITION_MOUNT} ]
	then
		mkdir -p ${ROOT_PARTITION_MOUNT}
	fi
	mount -t ext4 "$1" ${ROOT_PARTITION_MOUNT}

	arrowecho "Copying image and kernel binary to partition"

	# Get files needed to boot in place.
	cp ${MOBY_SRC_ROOT}/syslinux.cfg ${ROOT_PARTITION_MOUNT}

	# Increment number in moby src root to ensure beta partition boots to
	# correct drive
	sed -i 's#/dev/sda1#/dev/sda2#' ${MOBY_SRC_ROOT}/syslinux.cfg
	sed -i 's#Ishmael#Ahab#' ${MOBY_SRC_ROOT}/syslinux.cfg

	cat ${ROOT_PARTITION_MOUNT}/syslinux.cfg
	cp ${MOBY_SRC_ROOT}/vmlinuz64 ${ROOT_PARTITION_MOUNT}
	cp ${MOBY_SRC_ROOT}/initrd.img ${ROOT_PARTITION_MOUNT}

	# From http://www.syslinux.org/wiki/index.php?title=EXTLINUX:
	#
	# "Note that EXTLINUX installs in the filesystem partition like a
	# well-behaved bootloader :). Thus, it needs a master boot record in the
	# partition table; the mbr.bin shipped with SYSLINUX should work well."

	# Thus, this step installs syslinux on the mounted filesystem (partition
	# 1).
	arrowecho "Installing syslinux to partition"
	extlinux --install ${ROOT_PARTITION_MOUNT} 

	umount ${ROOT_PARTITION_MOUNT}
}
