#!/bin/sh

set -ex

docker version
docker info
docker ps
DOCKER_CONTENT_TRUST=1 docker pull alpine:3.5
docker run --rm alpine true
docker pull armhf/alpine
docker run --rm armhf/alpine uname -a
docker swarm init
docker run mobylinux/check-config:bc2b57a0770129c75a6676ae0c944ece1d50cc3f
