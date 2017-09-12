#!/bin/sh

set -e

: ${KUBE_MASTER_VCPUS:=2}
: ${KUBE_MASTER_MEM:=1024}
: ${KUBE_MASTER_DISK:=4G}

: ${KUBE_NODE_VCPUS:=2}
: ${KUBE_NODE_MEM:=4096}
: ${KUBE_NODE_DISK:=8G}

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

    : ${KUBE_VCPUS:=$KUBE_MASTER_VCPUS}
    : ${KUBE_MEM:=$KUBE_MASTER_MEM}
    : ${KUBE_DISK:=$KUBE_MASTER_DISK}
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

    : ${KUBE_VCPUS:=$KUBE_NODE_VCPUS}
    : ${KUBE_MEM:=$KUBE_NODE_MEM}
    : ${KUBE_DISK:=$KUBE_NODE_DISK}
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
