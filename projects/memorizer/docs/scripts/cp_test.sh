#!/bin/sh

cd /sys/kernel/debug/memorizer
echo 1 > clear_object_list
echo 1 > clear_printed_list
echo 1 > memorizer_enabled
echo 1 > memorizer_log_access

cd /root


cp -R /mnt/host/src/repos/linuxkit /root
echo "Done Copying"
cd /sys/kernel/debug/memorizer
echo 0 > memorizer_log_access
echo 0 > memorizer_enabled

cd /root
./userApp p > outputMap
