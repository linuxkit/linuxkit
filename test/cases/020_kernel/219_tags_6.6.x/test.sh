#!/bin/sh
# SUMMARY: Test existence and correctness of kernel builder tag, label and file
# LABELS:
# REPEAT:

set -e

KERNEL=linuxkit/kernel:6.6.13-4f0f536b9a057590102379043a0815d2f0e28209

# just include the common test
. ../tags.sh
