#!/bin/sh
# SUMMARY: Test existence and correctness of kernel builder tag, label and file
# LABELS:
# REPEAT:

set -e

KERNEL=linuxkit/kernel:6.6.71-bbe6930a9db6e1062d92357df654acc1d2d5832f

# just include the common test
. ../tags.sh
