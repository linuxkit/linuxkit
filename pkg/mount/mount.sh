#!/bin/sh

set -x

# read optional device indentifier
if [ $# -eq 2 ]
then
	DEV_ID="$1"
	shift
fi

MOUNTPOINT="$1"

[ -z "$MOUNTPOINT" ] && echo "No mountpoint specified" && exit 1

mkdir -p "$MOUNTPOINT"

mount_drive()
{
	# if explicit device identifier was given
	if [ -n "$DEV_ID" ]
	then
		mount_specified_drive && return
	else
		mount_detected_drive && return
	fi

	echo "WARNING: Failed to mount a persistent volume at $MOUNTPOINT (is there one?)"
}

mount_specified_drive()
{
	# if explicit device identifier was given
	if [ -n "$DEV_ID" ]
	then
		if [ -n "$(echo $DEV_ID | grep -E '^(LABEL|UUID)=')" ]
		then
			# identifier is LABEL or UUID
			DEV=$(findfs $DEV_ID)
			mount "$DEV" "$MOUNTPOINT" && return
		elif [ -b "$DEV_ID" ]
		then
			# identifier is a block-device
			DEV="$DEV_ID"
			mount "$DEV" "$MOUNTPOINT" && return
		fi
		echo "Warning: Unknown Device identifier provided: $DEV_ID"
	fi

	# unable to mount specified device
	return 1
}

mount_detected_drive()
{
	# TODO fix for multiple disks, cdroms etc
	DEVS="$(find /dev -maxdepth 1 -type b ! -name 'loop*' ! -name 'nbd*' | grep -v '[0-9]$' | sed 's@.*/dev/@@' | sort)"

	for DEV in $DEVS
	do
		DRIVE="/dev/${DEV}"

		# see if it has a partition table
		if sfdisk -d "${DRIVE}" >/dev/null 2>/dev/null
		then
			# 83 is Linux partition identifier
			DATA=$(sfdisk -J "$DRIVE" | jq -e -r '.partitiontable.partitions | map(select(.type=="83")) | .[0].node')
			if [ $? -eq 0 ]
			then
				mount "$DATA" "$MOUNTPOINT" && return
			fi
		fi
	done

	# unable to detect and mount device
	return 1
}

mount_drive
