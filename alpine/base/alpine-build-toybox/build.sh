#!/bin/sh

FILES=$@
make $FILES > /dev/null
[ $# -eq 0 ] && FILES=toybox
# TODO symlinks if just use toybox
mkdir -p /out/bin
mv $FILES /out/bin
printf "FROM scratch\nCOPY bin/ bin/\n" > /out/Dockerfile
cd /out
tar cf - .
