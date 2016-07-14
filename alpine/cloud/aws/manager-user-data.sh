#!/bin/sh

METADATA=http://169.254.169.254/latest/meta-data

# TODO: This dial retry loop should be handled by openrc maybe? (or by docker
# service)
docker swarm init \
	--secret "" \
	--auto-accept manager \
	--auto-accept worker \
	--listen-addr $(wget -qO- ${METADATA}/local-ipv4 | sed 's/http:\/\///'):4500 \
	>>/var/log/docker-swarm.log 2>&1
exit 0

exit 1
