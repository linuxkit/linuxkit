#!/bin/sh

set -e

help()
{
    echo "Usage: $0 [options] <dm_name> <device>"
    echo
    echo "Options:"
    echo "    -l|--luks       Use LUKS extension"
    echo "    -k|--key-file   Name of the key file, default: key"
    echo "    <dm_name>       Name of the device mapper file, the encrypted device will become available under /dev/mapper/<dm_name>"
    echo "    <device>        The encrypted device (e.g. /dev/sda1, /dev/loop0, etc)"
    echo
}

luks=false
key_file="key"

O=`getopt -l key-file:luks,help -- k:lh "$@"` || exit 1
eval set -- "$O"
while true; do
    case "$1" in
        -l|--luks)      luks=true; shift;;
        -k|--key-file)  key_file=$2; shift 2;;
        -h|--help)      help; exit 0;;
        --)             shift; break;;
        *)              echo "Unknown option $1"; help; exit 1;;
        esac
done

if [ -z "$1" ]; then
    echo "Missing argument <dm_name>"
    help
    exit 1
fi

if [ -z "$2" ]; then
    echo "Missing argument <device>"
    help
    exit 1
fi

dm_name=$1
device=$2
dmdev_name="/dev/mapper/$dm_name"
cipher="aes-cbc-essiv:sha256"

case "$key_file" in
    /*) ;;
    *)  key_file="/etc/dm-crypt/$key_file" ;;
esac

if [ ! -f "$key_file" ]; then
    echo "Couldn't find encryption keyfile $key_file!"
    exit 1
fi

if [ ! -d "/run/cryptsetup" ]; then
    echo "Creating cryptsetup lock directory"
    mkdir /run/cryptsetup
fi

if [ $luks = true ]; then
    echo "Creating dm-crypt LUKS mapping for $device under $dmdev_name"
    if ! cryptsetup isLuks $device; then
        echo "Device $device doesn't seem to have a valid LUKS setup so one will be created."
        cryptsetup --key-file "$key_file" --cipher "$cipher" luksFormat "$device"
    fi
    cryptsetup --key-file "$key_file" luksOpen "$device" "$dm_name"
else
    echo "Creating dm-crypt mapping for $device under $dmdev_name"
    cryptsetup --key-file "$key_file" --cipher "$cipher" create "$dm_name" "$device"
fi

o=`blkid $dmdev_name`
if [ -z "$o" ]; then
	echo "Device $dmdev_name doesn't seem to contain a filesystem, creating one."
    # dd will write the device until it's full and then return with an error because "no space left"
    dd if=/dev/zero of="$dmdev_name" || true
    mkfs.ext4 "$dmdev_name"
else
	echo "Device $dmdev_name seems to contain filesystem: $o"
fi
