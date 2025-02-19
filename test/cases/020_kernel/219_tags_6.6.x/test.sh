#!/bin/sh
# SUMMARY: Test existence and correctness of kernel builder tag, label and file
# LABELS:
# REPEAT:

set -e

KERNEL=linuxkit/kernel:6.6.71-819af9d59279506dd2994e7aea1cbbaaebfdb0a2

# just include the common test
. ../tags.sh
