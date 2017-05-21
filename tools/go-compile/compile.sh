#!/bin/sh

# This is designed to compile a single package to a single binary
# so it makes some assumptions about things to simplify config
# to output a single binary (in a tarball) just use -o file

set -e

usage() {
	echo "Usage: -o file"
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
	--package)
		package="$2"
		shift
	;;
	--ldflags)
		ldflags="$2"
		shift
	;;
	--clone-path)
		clonepath="$2"
		shift
	;;
	--clone)
		clone="$2"
		shift
	;;
	--commit)
		commit="$2"
		shift
	;;
	*)
		echo "Unknown option $1"
		exit 1
	esac
	shift
done

[ $# -gt 0 ] && usage
[ -z "$out" ] && usage

[ -z "$package" ] && package=$(basename "$out")

dir="$GOPATH/src/$package"

if [ -z "$clone" ]
then
	mkdir -p "$dir"
	cd "$dir"
	# untar input
	tar xf -
else
	mkdir -p "$GOPATH/src/$clonepath"
	cd "$GOPATH/src/$clonepath"
	git clone "$clone" .
	[ ! -z "$commit" ] && git checkout "$commit"
	mkdir -p "$dir"
	cd "$dir"
fi

cd "$dir"

# lint before building
>&2 echo "gofmt..."
test -z $(gofmt -s -l .| grep -v .pb. | grep -v vendor/ | tee /dev/stderr)

>&2 echo "govet..."
test -z $(GOOS=linux go tool vet -printf=false . 2>&1 | grep -v vendor/ | tee /dev/stderr)

>&2 echo "golint..."
test -z $(find . -type f -name "*.go" -not -path "*/vendor/*" -not -name "*.pb.*" -exec golint {} \; | tee /dev/stderr)

>&2 echo "ineffassign..."
test -z $(find . -type f -name "*.go" -not -path "*/vendor/*" -not -name "*.pb.*" -exec ineffassign {} \; | tee /dev/stderr)

>&2 echo "go build..."

if [ "$GOOS" = "darwin" -o "$GOOS" = "windows" ]
then
	if [ -z "$ldflags" ]
	then
		go build -o $out "$package"
	else
		go build -o $out -ldflags "${ldflags}" "$package"
	fi
else
	go build -o $out -buildmode pie -ldflags "-s -w ${ldflags} -extldflags \"-static\"" "$package"
fi

tar cf - $out
