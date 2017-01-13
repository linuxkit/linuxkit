#!/bin/sh

# This is designed to compile a single package to a single binary
# so it makes some assumptions about things to simplify config
# to output a single binary (in a tarball) just use -o file
# use --docker to output a tarball for input to docker build -

set -e

usage() {
	echo "Usage: -o file [--docker]"
	exit 1
}

[ $# = 0 ] && usage

while [ $# -gt 1 ]
do
	flag="$1"
	case "$flag" in
	-o)
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

[ $# -gt 0 ] && [ $1 = "--docker" ] && DOCKER=1 && shift

[ $# -gt 0 ] && usage
[ -z "$out" ] && usage

package=$(basename "$out")

dir="$GOPATH/src/$package"

mkdir -p $dir

# untar input
tar xf - -C $dir

/usr/bin/lint.sh $dir

go build -o $out --ldflags '-extldflags "-fno-PIC -static"' "$package"

if [ -z "$DOCKER" ]
then
	tar cf - $out
	exit 0
fi

printf "FROM scratch\nCOPY $out $out\nENTRYPOINT [\"$out\"]\n" > Dockerfile

tar cf - Dockerfile $out
