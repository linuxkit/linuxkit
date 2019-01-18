#!/bin/sh

set -e

help()
{
    echo "Usage: $0 [options] <file>"
    echo
    echo "Options:"
    echo "    -c, --create          Create <file> if not present, default: false"
    echo "    -s, --size NUM        Size of <file> in MiB if it gets created, default: 10"
    echo "    -d, --dev DEVICE      Use DEVICE as loop device, default: /dev/loop0"
    echo
}

create=false
size_mib=10
loop_device="/dev/loop0"

O=`getopt -l create,size:,dev:,help -- cs:d:h "$@"` || exit 1
eval set -- "$O"
while true; do
    case "$1" in
        -c|--create)   create=true; shift;;
        -s|--size)     size_mib=$2; shift 2;;
        -d|--dev)      loop_device=$2; shift 2;;
        -h|--help)     help; exit 0;;
        --)            shift; break;;
        *)             echo "Unknown option $1"; help; exit 1;;
	esac
done

if [ -z "$1" ]; then
    echo "Missing argument <file>"
    help
    exit 1
fi

container_file=$1

if [ ! -b "$loop_device" ]; then
    echo "Loop device $loop_device doesn't exist! Did you forget to bind-mount '/dev'?"
    exit 2
fi

if [ ! -f "$container_file" ]; then
    if [ $create = true ]; then
        echo "File $container_file not found, creating new one of size $size_mib MiB"
        dd if="/dev/zero" of="$container_file" bs=1M count=$size_mib
    else
        echo "File $container_file not found. Please specify --create or ensure it's present."
        exit 2
    fi
fi

echo "Associating file $container_file with loop device $loop_device"
losetup "$loop_device" "$container_file"
