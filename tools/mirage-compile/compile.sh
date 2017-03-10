#!/bin/sh

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
	*)
		echo "Unknown option $1"
		exit 1
	esac
	shift
done

[ -z "$out" ] && usage

package=$(basename "$out")

dir="/src"

# untar input
tar xf - -C $dir

(
    cd $dir
    opam config exec -- mirage configure -o $out -t unix
    opam config exec -- make depend
    opam config exec -- make
    mv $(readlink $out) $out
) > /src/logs 2>&1

cd $dir && tar -cf - $out

exit 0
