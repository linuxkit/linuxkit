#!/bin/bash

set -e

function die {
	echo >&2 "$@"
	exit 1
}

img=/usr/share/clear-containers/clear-containers.img
img=$(readlink -f "$img")
img_size=$(du -b "${img}" |  awk '{print $1}')

kernel="$(pwd)/clear-containers-vmlinux"
kernel_cmdline_file="$(pwd)/clear-containers-cmdline"
[ -f "${img}" ] || die "Image s required"
[ -f "${kernel}" ] || die "Kernel is required"
[ -f ${kernel_cmdline_file} ] || \
	die "Kernel cmdline file is required"

kernel_cmdline=$(cat "$kernel_cmdline_file")

cmd="/usr/bin/qemu-lite-system-x86_64"
cmd="$cmd -machine pc-lite,accel=kvm,kernel_irqchip,nvdimm"
cmd="$cmd -device nvdimm,memdev=mem0,id=nv0"
#image
cmd="$cmd -object memory-backend-file,id=mem0,mem-path=${img},size=${img_size}"
#memory
cmd="$cmd -m 2G,slots=2,maxmem=3G"
#kernel
cmd="$cmd -kernel ${kernel}"
cmd="$cmd -append '${kernel_cmdline}'"
#cpu
cmd="$cmd -smp 2,sockets=1,cores=2,threads=1"
cmd="$cmd -cpu host"
#clock
cmd="$cmd -rtc base=utc,driftfix=slew"
cmd="$cmd -no-user-config"
cmd="$cmd -nodefaults"
cmd="$cmd -global"
cmd="$cmd kvm-pit.lost_tick_policy=discard"
#console
cmd="$cmd -device virtio-serial-pci,id=virtio-serial0"
cmd="$cmd -chardev stdio,id=charconsole0,signal=off"
cmd="$cmd -device virtconsole,chardev=charconsole0,id=console0"
cmd="$cmd -nographic"
cmd="$cmd -vga none"

eval "$cmd"
