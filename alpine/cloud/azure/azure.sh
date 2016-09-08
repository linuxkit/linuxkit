#!/bin/sh

if [ "$1" = "tarout" ]
then
	tar --directory /tmp -cf - -S mobylinux.vhd
else
	./bake-azure.sh "$@" 1>&2
	if [ "$1" = "uploadvhd" ]
	then
		cat vhd_blob_url.out
	fi
fi
