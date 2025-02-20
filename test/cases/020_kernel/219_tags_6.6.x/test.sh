#!/bin/sh
# SUMMARY: Test existence and correctness of kernel builder tag, label and file
# LABELS:
# REPEAT:

set -e

KERNEL=linuxkit/kernel:6.6.71-fd482968e7997f0d956a5fd823dfdf5525841938

# just include the common test
. ../tags.sh
