#!/bin/sh
# SUMMARY: Test existence and correctness of kernel builder tag, label and file
# LABELS:
# REPEAT:

set -e

KERNEL=linuxkit/kernel:6.12.59-0ef72d722190ecfe0b3b37711f9a871a696e301a

# just include the common test
. ../tags.sh
