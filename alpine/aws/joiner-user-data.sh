#!/bin/sh

function logcmd () {
    "$@" | awk -v timestamp="$(date) " '$0=timestamp$0' >>/var/log/docker-swarm.log
}

for i in $(seq 0 120); do
    logcmd docker swarm join {{MANAGER_IP}}:4242
    logcmd docker swarm info
    if [ $? -eq 0 ]; then
        exit 0
    fi
    logcmd "Join attempt failed, retrying"
    sleep 1
done
