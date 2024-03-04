#!/bin/sh
# SUMMARY: Test existence and correctness of kernel builder tag, label and file
# LABELS:
# REPEAT:

set -e

KERNEL=linuxkit/kernel:5.4.172-9005a97e2b2cba68b4374092167b079a2874f66b

# just include the common test
. ../tags.sh
