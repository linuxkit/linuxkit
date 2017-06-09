#!/bin/bash -eu
[ "${#@}" -gt 1 ] || (echo "Usage: ${0} <node> <join_args>" ; exit 1)
name="node-${1}"
shift
disk="kube-${name}-disk.img"
set -x
rm -f "${disk}"
../../bin/linuxkit run -cpus 2 -mem 4096 -disk "${disk}",size=4G -data "${*}" kube-node
