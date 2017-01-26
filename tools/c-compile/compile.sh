#!/bin/sh

# This is designed to compile a single package to a single binary
# so it makes some assumptions about things to simplify config
# to output a single binary (in a tarball) just use -o file
# use --docker to output a tarball for input to docker build -

set -e

usage() {
	echo "Usage: -o file"
	exit 1
}

[ $# = 0 ] && usage

while [ $# -gt 0 ]
do
	flag="$1"
	case "$flag" in
	-o)
		[ $# -eq 1 ] && usage
		out="$2"
		mkdir -p "$(dirname $2)"
		shift
	;;
	-l*)
		LIBS="$LIBS $1"
		shift
	;;
	*)
		echo "Unknown option $1"
		exit 1
	esac
	shift
done

[ -z "$out" ] && usage

package=$(basename "$out")

dir="/src/$package"

mkdir -p $dir

# untar input
tar xf - -C $dir

(
	cd $dir
	CFILES=$(find . -name '*.c')
	cc -static -O2 -Wall -Werror -o ../../$out $CFILES $LIBS
)

tar cf - $out
exit 0
