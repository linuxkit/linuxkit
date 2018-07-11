#!/bin/sh

for i in $(seq 1 20); do
	if [ -e /var/log/fill-the-logs.out.log.0 ]; then
		printf "logwrite test suite PASSED\n" > /dev/console
		/sbin/poweroff -f
	fi
	sleep 1
done

printf "logwrite test suite FAILED\n" > /dev/console
echo "contents of /var/log:" > /dev/console
ls -l /var/log > /dev/console
/sbin/poweroff -f
