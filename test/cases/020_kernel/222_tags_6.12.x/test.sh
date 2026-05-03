#!/bin/sh
# SUMMARY: Test existence and correctness of kernel builder tag, label and file
# LABELS:
# REPEAT:

set -e

KERNEL=linuxkit/kernel:6.12.85-1e7c40b4a4edb654e920e399abf64b11c1b86f45

# just include the common test
. ../tags.sh
