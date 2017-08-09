#!/bin/sh

set -e

: ${KUBE_PORT_BASE:=2222}
: ${KUBE_VCPUS:=2}
: ${KUBE_MEM:=1024}
: ${KUBE_DISK:=4G}
: ${KUBE_NETWORKING:=default}
: ${KUBE_RUN_ARGS:=}
: ${KUBE_EFI:=}

[ "$(uname -s)" = "Darwin" ] && KUBE_EFI=1

suffix=".iso"
[ -n "${KUBE_EFI}" ] && suffix="-efi.iso" && uefi="--uefi"

if [ $# -eq 0 ] ; then
    img="kube-master"
    data=""
    state="kube-master-state"
elif [ $# -gt 1 ] ; then
    case $1 in
	''|*[!0-9]*)
	    echo "Node number must be a number"
	    exit 1
	    ;;
	0)
	    echo "Node number must be greater than 0"
	    exit 1
	    ;;
	*) ;;
    esac
    img="kube-node"
    name="node-${1}"
    shift
    data="${*}"
    state="kube-${name}-state"
else
    echo "Usage:"
    echo " - Boot master:"
    echo "   ${0}"
    echo " - Boot node:"
    echo "   ${0} <node> <join_args>"
    exit 1
fi
set -x
rm -rf "${state}"
linuxkit run ${KUBE_RUN_ARGS} -networking ${KUBE_NETWORKING} -cpus ${KUBE_VCPUS} -mem ${KUBE_MEM} -state "${state}" -disk size=${KUBE_DISK} -data "${data}" ${uefi} "${img}${suffix}"
