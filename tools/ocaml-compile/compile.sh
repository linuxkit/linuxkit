#!/bin/sh

# This is designed to compile a single package to a single binary
# so it makes some assumptions about things to simplify config
# to output a single binary (in a tarball) just use -o file
# use --docker to output a tarball for input to docker build -

set -ex

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
		echo $PWD
		mkdir -p "$(dirname $2)"
		shift
	;;
	--libs)
		[ $# -eq 1 ] && usage
		LIBS="$LIBS -package $2"
		shift
	;;
	--pkgs)
		[ $# -eq 1 ] && usage
		PKGS="$PKGS $2"
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

OARGS="-g -warn-error +1..49+60 -w A-4-41-44-7 -short-paths -bin-annot -strict-sequence"

(
    cd $dir
    MLFILES=$(find . -name '*.ml')
    opam depext -uiy $PKGS
    opam config exec -- ocamlfind ocamlopt -o $out $MLFILES $OARGS -linkpkg $LIBS
    echo $(ls -la /tmp)
    tar -cf - $out
)

exit 0
