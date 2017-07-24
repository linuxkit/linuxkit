#!/bin/bash -eux
ssh="docker run --rm -ti \
  -v $HOME/.ssh/:/root/.ssh \
    jdeathe/centos-ssh \
	-o LogLevel=FATAL \
	-o StrictHostKeyChecking=no \
	-o UserKnownHostsFile=/dev/null \
	-o IdentitiesOnly=yes"
$ssh -t root@"$1" ctr exec --tty --exec-id ssh kubelet ash -l
