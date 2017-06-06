#!/bin/sh

set -e

PORTS=$(netstat -lntup)
LINES=$(echo "${PORTS}" | wc -l)
if [ $((LINES > 2)) -ne 0 ]
then
	echo "Ports test case FAILED"
	echo "${PORTS}"
	exit 1
fi
echo "Ports test case PASSED"
