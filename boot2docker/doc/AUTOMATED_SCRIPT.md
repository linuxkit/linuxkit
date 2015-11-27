Automated script
===========

##  Overview
The document describes the "automated script" functionality and some use cases. I assume 
you have basic knowledge of iPxe/pxe booting. Please see the following link for more documentaion regarding iPxe:
http://ipxe.org/

## Summary
Automated script allows you to run an arbitrary script as a boot parameter. The script 
parses `/proc/cmdline` for `script` variable and download the script if its either
an ftp or http uri. This method is used by Archlinux to install unattendedly. Please see the
following link for reference:

https://projects.archlinux.org/archiso.git/tree/configs/releng/airootfs/root/.automated_script.sh 

## Usecase
Lets say you want to run boot2docker on multiple physical machines and you want run hashicorp/consul
on all machines. You boot the machine via ipxe and add `script` as a parameter. The script you
use could take information from the host as an identifier and announce itself using that information.

## Extracting initrd and vmlinuz64
You mount the iso and the files are located in `/boot`.

## Extracting boot parameters
Extracting the parameters and adjusting these will help you boot and provision using iPxe. It will also
help you understand how boot2docker is being run currently.

```
root@896569876a97:/# cat /proc/cmdline
loglevel=3 user=docker console=ttyS0 console=tty0 noembed nomodeset norestore waitusb=10:LABEL=boot2docker-data base initrd=/boot/initrd.img BOOT_IMAGE=/boot/vmlinuz64
```

## iPxe example
```
#!ipxe

set script http://192.168.3.3:4321/repo/script.sh
set append loglevel=3 user=docker console=ttyS0 console=tty nomodeset norestore base script=${script}
set kernel http://192.168.3.3:4321/repo/vmlinuz64
set initrd http://192.168.3.3:4321/repo/initrd.img



imgfree
kernel ${kernel} ${append}
initrd ${initrd}
boot
```
