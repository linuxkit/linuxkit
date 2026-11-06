#!/bin/sh
# SUMMARY: Test existence and correctness of kernel builder tag, label and file
# LABELS:
# REPEAT:

set -e

KERNEL=linuxkit/kernel:6.12.115-87df45c48d5fa20d14e165fc52af1bf07b991a1a

# just include the common test
. ../tags.sh
