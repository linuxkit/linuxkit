#!/bin/sh
# SUMMARY: Test existence and correctness of kernel builder tag, label and file
# LABELS:
# REPEAT:

set -e

KERNEL=linuxkit/kernel:6.12.59-ad951380fc00653979e5b5b716a25f0c93d82c83

# just include the common test
. ../tags.sh
