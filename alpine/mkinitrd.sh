#!/bin/sh

find / -xdev | cpio -H newc -o > /export/initrd.img
