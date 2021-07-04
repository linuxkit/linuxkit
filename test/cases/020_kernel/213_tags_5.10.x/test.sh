#!/bin/sh
# SUMMARY: Test existence and correctness of kernel builder tag, label and file
# LABELS:
# REPEAT:

set -e

KERNEL=linuxkit/kernel:5.10.47

# just include the common test
. ../tags.sh
