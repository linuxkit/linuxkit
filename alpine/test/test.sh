#!/bin/sh

set -ex

docker version
docker info
docker ps
DOCKER_CONTENT_TRUST=1 docker pull alpine
docker run --rm alpine true
docker pull armhf/alpine
docker run --rm armhf/alpine uname -a
docker swarm init
docker run mobylinux/check-config@sha256:4282f589d5a72004c3991c0412e45ba0ab6bb8c0c7d97dc40dabc828700e99ab
