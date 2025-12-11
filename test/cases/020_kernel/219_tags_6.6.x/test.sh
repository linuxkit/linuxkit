#!/bin/sh
# SUMMARY: Test existence and correctness of kernel builder tag, label and file
# LABELS:
# REPEAT:

set -e

KERNEL=linuxkit/kernel:6.6.71-fac3a2ad495a23a50c8d2d7f173ecc7145049c29

# just include the common test
. ../tags.sh
