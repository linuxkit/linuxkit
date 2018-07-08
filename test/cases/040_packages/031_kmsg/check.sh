#!/bin/sh

for i in $(seq 1 20); do
	# Look for a common kernel log message
	if grep "SCSI subsystem initialized" /var/log/kmsg.out 2>/dev/null; then
		printf "kmsg test suite PASSED\n" > /dev/console
		/sbin/poweroff -f
	fi
	sleep 1
done

printf "kmsg test suite FAILED\n" > /dev/console
echo "contents of /var/log/kmsg.out:" > /dev/console
cat /var/log/kmsg.out > /dev/console
/sbin/poweroff -f
