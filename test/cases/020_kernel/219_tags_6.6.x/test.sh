#!/bin/sh
# SUMMARY: Test existence and correctness of kernel builder tag, label and file
# LABELS:
# REPEAT:

set -e

KERNEL=linuxkit/kernel:6.6.71-a52f587ad371e287eaf4790265b90f82def98994

# just include the common test
. ../tags.sh
