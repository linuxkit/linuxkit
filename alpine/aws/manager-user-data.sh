#!/bin/sh

# TODO: This should be handled by openrc maybe? (or by docker service)
for i in $(seq 1 20); do
    if [ -S /var/run/docker.sock ]; then
        docker swarm create >>/var/log/docker-swarm.log 2>&1
        exit 0
    fi
    sleep 1
done

exit 1
