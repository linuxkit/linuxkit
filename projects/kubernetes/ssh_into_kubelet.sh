#!/bin/bash -eux
./ssh.sh -t root@"$1" ctr exec --tty --exec-id ssh kubelet ash -l
