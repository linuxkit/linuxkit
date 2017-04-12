#!/bin/sh

sh runltplite.sh -p -l /ltp.log
cat /ltp.log

baseline="$(cat /etc/ltp/baseline)"
failures="$( grep "Total Failures" /ltp.log | awk '{print $3}')"

if [ $((failures <= baseline)) -ne 0 ]
then
	printf "LTP test suite PASSED\n"
else
	printf "LTP test suite FAILED\n"
	exit 1
fi
