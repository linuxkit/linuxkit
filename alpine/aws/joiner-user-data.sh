#!/bin/sh

logcmd() {
    "$@" 2>&1 | awk -v timestamp="$(date) " '$0=timestamp$0' >>/var/log/docker-swarm.log
}

for i in $(seq 1 20); do
    if [ -S /var/run/docker.sock ]; then
        logcmd docker swarm join {{MANAGER_IP}}:4500
        logcmd docker swarm info
        exit 0
    fi
    sleep 1
done

exit 1
