#!/bin/sh

# load the audit rules into the kernel
auditctl -R /etc/audit/audit.rules
exec /sbin/auditd -f
