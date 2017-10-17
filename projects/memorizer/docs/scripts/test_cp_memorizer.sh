#!/bin/sh

KDIR=/sys/kernel/debug/memorizer
UDIR=/mnt/host/src/repos/linux-slice/scripts
cd $WDIR
echo 1 > clear_object_list
echo 1 > clear_printed_list
echo 1 > memorizer_enabled
echo 1 > memorizer_log_access


cp -R /mnt/host/src/repos/linuxkit /root
cd /root
./userApp

cd $WDIR
echo 0 > memorizer_enabled
echo 0 > memorizer_log_access
