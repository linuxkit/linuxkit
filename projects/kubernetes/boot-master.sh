#!/bin/bash -eu
disk="kube-master-disk.img"
set -x
rm -f "${disk}"
../../bin/moby run hyperkit -cpus 2 -mem 6144 -disk-size 6144 kube-master
