#!/bin/sh
cd /sys/kernel/debug/memorizer
echo 1 > clear_object_list
echo 1 > clear_printed_list
echo 1 > memorizer_enabled
echo 1 > memorizer_log_access
