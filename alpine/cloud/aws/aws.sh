#!/bin/sh

./bake-ami.sh "$@" 1>&2
if [ "$1" = "bake" ]
then
	cat /build/ami_id.out
fi
