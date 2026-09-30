#!/bin/sh

for i in $(seq 1 20); do
	if grep "Init complete" /var/log/auditd.log 2>/dev/null; then
		printf "auditd test suite PASSED\n" > /dev/console
		/sbin/poweroff -f
	fi
	sleep 1
done

printf "auditd test suite FAILED\n" > /dev/console
echo "contents of /var/log/auditd.log:" > /dev/console
cat /var/log/auditd.log > /dev/console
/sbin/poweroff -f
