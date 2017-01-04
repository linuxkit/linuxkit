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
docker run mobylinux/check-kernel-config@sha256:beabc0fd77bb9562a03104eecb34286d5aa695896e0d3e56b36876b24d2a9126
