#!/bin/sh
# SUMMARY: Test existence and correctness of kernel builder tag, label and file
# LABELS:
# REPEAT:

set -e

KERNEL=linuxkit/kernel:6.12.59-d8fd304753118b757f473dd5fd087f33c238bb37

# just include the common test
. ../tags.sh
