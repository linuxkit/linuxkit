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

Available modes: --hash, --image, --pkg

Replace by hash:
        $0 --hash <OLD> <NEW>
        Example: $0 --hash 8675309abcdefg abcdef567899
           Will replace all instances of 8675309abcdefg with abcdef567899

Replace by image:
        $0 --image <IMAGE> <NEW>
        Example: $0 --image linuxkit/foo abcdef567899
           Will tag all instances of linuxkit/foo with abcdef567899

        $0 --image <IMAGE>:<NEW> is accepted as a convenient shortcut for cutting
        and pasting e.g.the output of linuxkit pkg show-tag

Replace by pkg directory:
        $0 --pkg <PATH/TO/PKG> <NEW>
        Example: $0 --pkg ./pkg/xen-tools
           Will use linuxkit pkg show-tag on the directory ./pkg/xen-tools, and then
     tag all instances with the result

EOF
}

updateImage() {
        local image
        local hash

        case $# in
        1)
                image=${1%:*}
                hash=${1#*:}
                ;;
        2)
                image=$1
                hash=$2
                ;;
        esac
        git grep -E -l "[[:space:]]$image:" -- '*.yml' '*.yaml' '*.yml.in' '*.yaml.in' '*/Dockerfile' '*/Makefile' | grep -v /vendor/ | xargs sed -i.bak -E -e "s,([[:space:]])($image):([^[:space:]]+), $image:$hash,g"
}

# backwards compatibility
if [ $# -eq 2 -a -n "${1%--*}" ]; then
  set -- "--hash" "$1" "$2"
fi

# which mode?
mode=$1
shift

case "${mode}" in
--hash)
        if [ $# -ne 2 ] ; then
                usage
                exit 1
        fi
        old=$1
        new=$2
        git grep -E -l "\b($old)([[:space:]].*)?$" -- '*.yml' '*.yaml' '*.yml.in' '*.yaml.in' '*/Dockerfile' '*/Makefile' | grep -v /vendor/ | while read -r file; do sed -ri.bak -e "s,$old,$new,g" "$file"; done
        ;;
--image)
	if [ $# -lt 1 ] ; then
		usage
		exit 1
	fi

        updateImage $@
        ;;
--pkg)
        if [ $# -ne 1 ]; then
                usage
                exit 1
        fi
	if [ ! -d "$1" ]; then
		echo "Directory '$1' does not exist"
		usage
		exit 1
	fi
        tag=$(linuxkit pkg show-tag $1)
        updateImage ${tag}
        ;;
*)
	echo "Unknown mode $1"
	usage
	exit 1
	;;
esac

find . -name '*.bak' | xargs rm
