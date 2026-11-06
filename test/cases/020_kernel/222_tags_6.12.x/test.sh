#!/bin/sh
# SUMMARY: Test existence and correctness of kernel builder tag, label and file
# LABELS:
# REPEAT:

set -e

KERNEL=linuxkit/kernel:6.12.115-08878abbd9f43fda1eb037e42f7417ea0a502635

# just include the common test
. ../tags.sh
