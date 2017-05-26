#!/bin/sh
# SUMMARY: Check that the kernel+initrd image boots on hyperkit
# LABELS:

set -e
set -x

# Source libraries. Uncomment if needed/defined
#. "${RT_LIB}"
. "${RT_PROJECT_ROOT}/_lib/lib.sh"

NAME=hyperkit-kernel

clean_up() {
	echo $(pwd)
	# remove any files, containers, images etc
	rm -rf "${NAME}"* || true
}
trap clean_up EXIT

moby build -output kernel+initrd -name "${NAME}" test.yml
[ -f "${NAME}-kernel" ] || exit 1
[ -f "${NAME}-initrd.img" ] || exit 1
[ -f "${NAME}-cmdline" ]|| exit 1
./test.exp
exit 0
