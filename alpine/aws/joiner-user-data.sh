#!/bin/sh

logcmd() {
    "$@" | awk -v timestamp="$(date) " '$0=timestamp$0' >>/var/log/docker-swarm.log 2>&1
}

logcmd docker swarm join {{MANAGER_IP}}:4500
logcmd docker swarm info
