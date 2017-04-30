#!/bin/sh

set -e

while [ $# -ge 1 ]; do
	key="$1"

	case $key in
		--debug)
			set -x
			;;
		--path)
			path="$2"
			shift # past argument
			;;
		--size)
			size="$2"
			shift # past argument
			;;
		--encrypt)
			ENCRYPT=true
			;;
		--condition)
			CONDITIONS="$CONDITIONS $2"
			shift
			;;
		*)
			echo "Unknown option passed to swapmaker: $key"      # unknown option
			exit 1
			;;
	esac
	shift # past argument or value
done

function disksize_to_count {
	local blocksize=$1
	local origsize=$2
	local ret
	case $origsize in
		*G)
			ret=$(( ${origsize%%G} * 1024 * 1024 * 1024 ))
			;;
		*M)
			ret=$(( ${origsize%%M} * 1024 * 1024 ))
			;;
		*K)
			ret=$(( ${origsize%%K} * 1024 ))
			;;
		*)
			ret=$origsize
			;;
	esac
	ret=$(( $ret / $blocksize ))
	echo $ret
}


## make sure path is valid
if [ -z "${path}" ]; then
	echo "swap: --file <path> must be defined"
	exit 1
fi
if [ "${path:0:5}" != "/var/" -o ${#path} -lt 6 ]; then
	echo "--file <path> option must be under /var"
	exit 1
fi
if [ -z "${size}" ]; then
	echo "swap: --size <size> must be defined"
	exit 1
fi



## check each of our conditions
for cond in $CONDITIONS; do
	# split the condition parts
	IFS=: read condtype arg1 arg2 arg3 arg4 <<EOF
$cond
EOF
	case $condtype in
		part)
			partition=$arg1
			required=$arg2
			# check that the path exists as its own mount point
			set +e
			grep -qs $partition /proc/mounts
			is_mnt=$?
			set -e
			if [ $is_mnt -ne 0 ]; then
				[ "$required" == "true" ] && exit 1
				exit 0
			fi
			;;
		partsize)
			partition=$arg1
			minsize=$arg2
			required=$arg3
			# check that the partition on which it exists has sufficient size
			partsize=$(df -k $partition | tail -1 | awk '{print $2}')
			partsize=$(( $partsize * 1024 ))
			# convert minsize to bytes
			minsize=$(disksize_to_count 1 $minsize)
			if [ $partsize -lt $minsize ]; then
				[ "$required" == "true" ] && exit 1
				exit 0
			fi
			;;
		*)
			echo "Unknown condition: $cond"
			exit 1
			;;
	esac
done
## if a condition failed:
### Required? exit 1
### Else? exit 0


## Allocate the file
dd if=/dev/zero of=$path bs=1024 count=$(disksize_to_count 1024 $size)
chmod 0600 $path

## was it encrypted? use cryptsetup and get the mapped device
if [ "$ENCRYPT" == "true" ]; then
	# might need
	#loop=$(losetup -f)
	#losetup ${loop} ${path}

	cryptsetup open --type plain --key-file /dev/urandom --key-size=256 --cipher=aes-cbc-essiv:sha256 --offset=0  ${path} swapfile
	SWAPDEV=/dev/mapper/swapfile
else
	SWAPDEV=$path
fi

## mkswap and swapon the device
/sbin/mkswap $SWAPDEV
/sbin/swapon $SWAPDEV
