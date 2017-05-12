#!/bin/bash -eux
./ssh.sh -t root@"$1" nsenter --mount --target 1 runc exec --tty kubelet ash -l
