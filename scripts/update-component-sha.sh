#!/bin/sh
set -e

##
#
# script to replace hashes in config files
# see usage() for usage and functionality
#

usage() {
    cat >&2 <<EOF
$0 --<mode> <how-to-find> <new-hash>

Available modes: --hash and --image

Replace by hash:
	$0 --hash <OLD> <NEW>
	Example: $0 8675309abcdefg abcdef567899
    	   Will replace all instances of 8675309abcdefg with abcdef567899

Replace by image: $0 --image <IMAGE> <NEW>
	$0 --image <IMAGE> <NEW>
	Example: $0 linuxkit/foo abcdef567899
	   Will tag all instances of linuxkit/foo with abcdef567899

By default, for convenience, if no mode is given (--image or --hash), the first method (--hash) is assumed. 
Thus the following are equivalent:
	$0 <OLD> <NEW>
	$0 --hash <OLD> <NEW>

EOF
}


# backwards compatibility
if [ $# -eq 2 ]; then
  set -- "--hash" "$1" "$2"
fi

# sufficient arguments
if [ $# -ne 3 ] ; then
    usage
    exit 1
fi


# which mode?
case "$1" in
    --hash)
        old=$2
        new=$3

        git grep -w -l "\b$old\b" | xargs sed -i.bak -e "s,$old,$new,g"
        ;;
    --image)
        image=$2
        hash=$3
        git grep -E -l "\b$image:" | xargs sed -i.bak -e "s,$image:[[:xdigit:]]"'\{40\}'",$image:$hash,g"
        ;;
    *)
        echo "Unknown mode $1"
        usage
        exit 1
        ;;
esac

find . -name '*.bak' | xargs rm
