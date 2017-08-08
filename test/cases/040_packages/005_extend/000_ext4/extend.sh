#!/bin/sh
set -ex
qemu-img resize -f qcow2 "$1" +256M
