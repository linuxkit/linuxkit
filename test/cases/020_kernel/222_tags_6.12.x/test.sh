#!/bin/sh
# SUMMARY: Test existence and correctness of kernel builder tag, label and file
# LABELS:
# REPEAT:

set -e

KERNEL=linuxkit/kernel:6.12.52-01ab8e22c88f25fc1bf4c354689f12797c213a86

# just include the common test
. ../tags.sh
