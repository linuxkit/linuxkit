#!/bin/sh

# Script to automate the creation of a VHD for Moby in Azure, and upload it to
# an Azure storage account (needed in order to launch it on Azure, or upload it
# to the Azure Marketplace).
#
# Usage: ./bake-azure.sh (intended to be invoked in a Docker container with
# specific properties, see the 'alpine' dir / Makefile)
#
# Parameters (override as environment variables):
#
# AZURE_STG_ACCOUNT_NAME: Name of the storage account to upload the VHD to.
#
# AZURE_STG_ACCOUNT_KEY: Key needed to access the storage account to upload the
# VHD.  This can be accessed in the storage account in the web portal.
#
# CONTAINER_NAME: Name of the container in the storage account to place the
# created VHD in.  "Container" here is NOT a Docker/Linux container, it is
# similar to "bucket" in AWS parlance.
#
# BLOBNAME: Name of the created VHD "blob".  e.g., "foobar-mobylinux.vhd"

set -e

PROVIDER="azure"

source "build-common.sh"

case "$1" in
	makeraw)
		RAW_IMAGE="${MOBY_SRC_ROOT}/mobylinux.img"

		if [ -f "${RAW_IMAGE}" ]
		then
			rm "${RAW_IMAGE}"
		fi

		arrowecho "Writing empty image file"
		dd if=/dev/zero of="${RAW_IMAGE}" count=0 bs=1 seek=30G

		arrowecho "Formatting image file for boot"
		format_on_device "${RAW_IMAGE}"

		arrowecho "Setting up loopback device"
		LOOPBACK_DEVICE="$(losetup -f --show ${RAW_IMAGE})"

		arrowecho "Loopback device is ${LOOPBACK_DEVICE}"

		arrowecho "Mapping partition"
		MAPPED_PARTITION="/dev/mapper/$(kpartx -av ${LOOPBACK_DEVICE} | cut -d' ' -f3)"
		arrowecho "Partition mapped at ${MAPPED_PARTITION}"

		arrowecho "Installing syslinux and dropping artifacts on partition..."
		configure_syslinux_on_device_partition "${LOOPBACK_DEVICE}" "${MAPPED_PARTITION}"

		arrowecho "Cleaning up..."
		kpartx -d "${LOOPBACK_DEVICE}"
		losetup -d "${LOOPBACK_DEVICE}"

		arrowecho "Finished making raw image file"
		;;

	uploadvhd)
		if [ -z "${AZURE_STG_ACCOUNT_KEY}" ]
		then
			errecho "Need to set AZURE_STG_ACCOUNT_KEY for the 'dockereditions' storage account."
			exit 1
		fi

		AZURE_STG_ACCOUNT_NAME=${AZURE_STG_ACCOUNT_NAME:-"dockereditions"}
		CONTAINER_NAME=${CONTAINER_NAME:-"mobylinux"}
		BLOBNAME=${BLOBNAME:-$(md5sum "${MOBY_SRC_ROOT}/mobylinux.vhd" | awk '{ print $1; }')-mobylinux.vhd}

		azure-vhd-utils-for-go upload \
			--localvhdpath "${MOBY_SRC_ROOT}/mobylinux.vhd" \
			--stgaccountname "${AZURE_STG_ACCOUNT_NAME}" \
			--stgaccountkey "${AZURE_STG_ACCOUNT_KEY}" \
			--containername "${CONTAINER_NAME}" \
			--blobname "${BLOBNAME}" \
			--overwrite

		arrowecho "VHD uploaded."
		arrowecho "https://${AZURE_STG_ACCOUNT_NAME}.blob.core.windows.net/${CONTAINER_NAME}/${BLOBNAME}"
		;;

	*)
		errecho "Invalid usage.  Syntax: ./bake-azure.sh [makeraw|uploadvhd]"
		exit 1
esac
