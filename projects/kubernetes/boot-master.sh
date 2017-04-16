#!/bin/bash -eu
disk="kube-master-disk.img"
set -x
rm -f "${disk}"
../../bin/moby run hyperkit -cpus 2 -mem 4096 -disk-size 4096 kube-master
