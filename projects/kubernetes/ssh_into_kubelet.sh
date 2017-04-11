#!/bin/bash -eux
./ssh.sh -t "$1" nsenter --mount --target 1 runc exec --tty kubelet bash -l
