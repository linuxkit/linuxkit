#!/bin/sh
# SUMMARY: Test existence and correctness of kernel builder tag, label and file
# LABELS:
# REPEAT:

set -e

KERNEL=linuxkit/kernel:6.12.52-d246bd1bb27e83cc43cc7bfa3576452a2edbde71

# just include the common test
. ../tags.sh
