#!/bin/bash -eu

sshopts="-o LogLevel=FATAL \
	 -o StrictHostKeyChecking=no \
	 -o UserKnownHostsFile=/dev/null \
	 -o IdentitiesOnly=yes"

case $(uname -s) in
    Linux)
	ssh=ssh
	;;
    *)
	ssh="docker run --rm -ti \
	  -v $HOME/.ssh/:/root/.ssh \
	    ijc25/alpine-ssh"
	;;
esac
$ssh $sshopts -t root@"$1" ctr tasks exec --tty --exec-id ssh kubelet ash -l
