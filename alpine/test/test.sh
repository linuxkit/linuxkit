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
docker run mobylinux/check-kernel-config@sha256:8c48a0f8456e1e5027eb2540974e6304d8cf58263c7553b61838cb627c4e790c
