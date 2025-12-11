#!/bin/sh
# SUMMARY: Test existence and correctness of kernel builder tag, label and file
# LABELS:
# REPEAT:

set -e

KERNEL=linuxkit/kernel:6.6.71-77b4563829536629b7d5768d4407453c26602d65

# just include the common test
. ../tags.sh
