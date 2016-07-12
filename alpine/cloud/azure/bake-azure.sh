#!/bin/bash

set -e

declare -xr PROVIDER="azure"

source "build-common.sh"

case "$1" in
    makeraw)
        RAW_IMAGE="${MOBY_SRC_ROOT}/mobylinux.img"

        if [[ -f "${RAW_IMAGE}" ]]; then
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
        if [[ -z "${AZURE_STG_ACCOUNT_KEY}" ]]; then
            errecho "Need to set AZURE_STG_ACCOUNT_KEY for the 'dockereditions' storage account."
            exit 1
        fi

        declare -xr AZURE_STG_ACCOUNT_NAME="dockereditions"
        declare -xr CONTAINER_NAME="mobylinux"
        declare -xr BLOBNAME="$(md5sum "${MOBY_SRC_ROOT}/mobylinux.vhd" | awk '{ print $1; }')-mobylinux.vhd"

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
