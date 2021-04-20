#!/bin/sh
function failed {
	printf "Kernel module test suite FAILED\n"
	/sbin/poweroff -f
}

uname -a
modinfo hello_world.ko || failed
insmod hello_world.ko || failed
[ -n "$(dmesg | grep -o 'Hello LinuxKit')" ] || failed
rmmod hello_world || failed

printf "Kernel module test suite PASSED\n"

/sbin/poweroff -f
