#!/bin/bash
# This can help get patches prefixed by the term of number.
#
# Just please list all patches in the file "series" orderly, and then run this
# script directly.
#
# $ cat series
# $ xxxx.patch
# $ yyyy.patch
# $ zzzz.patch
# $ ./prefix-with-number.sh
# $ ls -l
#   0001-xxxx.patch
#   0002-yyyy.patch
#   0003-zzzz.patch
#
i=0000
for line in `sed -e "s/#.*//g" series`; do
	i=$(expr $i + 1)
	a=$((10000+$i))
	mv $line ${a:1}-$line
done;
