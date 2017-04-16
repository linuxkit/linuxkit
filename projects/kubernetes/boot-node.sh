#!/bin/bash -eu
[ "${#@}" -gt 1 ] || (echo "Usage: ${0} <node> <join_args>" ; exit 1)
name="node-${1}"
shift
disk="kube-${name}-disk.img"
set -x
rm -f "${disk}"
../../bin/moby run hyperkit -cpus 2 -mem 4096 -disk-size 4096 -disk "${disk}" -data "${*}" kube-node
