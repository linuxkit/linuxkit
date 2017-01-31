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
docker run mobylinux/check-config:dc29b05bb5cca871f83421e4c4aaa8f5d3c682f4@sha256:5dcdf0e3386ed506a28a59191eaa1ea48261e15199fcbbe8caf8dc1889405b2d
docker run mobylinux/check-kernel-config:766a83e4b1831bef7f748071d0cd7715935d8be2@sha256:6821a7bce30bd013a6cc190d171228f9b02359e9c792858005f401ab15357575
cat /etc/moby
