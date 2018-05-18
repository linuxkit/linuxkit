#!/bin/sh

KEYS=$(find /etc/ssh -name 'ssh_host_*_key')
[ -z "$KEYS" ] && \
  ssh-keygen -t ed25519 -f /etc/ssh/ssh_host_ed25519_key -N '' && \
  ssh-keygen -t rsa -b 4096 -f /etc/ssh/ssh_host_rsa_key -N ''

exec /usr/sbin/sshd -D -e
