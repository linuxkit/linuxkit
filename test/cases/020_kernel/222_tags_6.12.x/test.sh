#!/bin/sh
# SUMMARY: Test existence and correctness of kernel builder tag, label and file
# LABELS:
# REPEAT:

set -e

KERNEL=linuxkit/kernel:6.12.59-367dfce300096c198d3333652b71beb68608a659

# just include the common test
. ../tags.sh
