#!/bin/sh

METADATA=http://169.254.169.254/latest/meta-data

# TODO: This dial retry loop should be handled by openrc maybe? (or by docker
# service)
for i in $(seq 1 20); do
    if [ -S /var/run/docker.sock ]; then
        docker swarm create \
            --listen-addr $(wget -qO- ${METADATA}/local-ipv4 | sed 's/http:\/\///'):4500 \
            >>/var/log/docker-swarm.log 2>&1
        exit 0
    fi
    sleep 1
done

exit 1
