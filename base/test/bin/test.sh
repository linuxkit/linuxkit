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
docker run mobylinux/check-kernel-config:b7616e925bc58ce9f9cc2b60009a95084ef4ca4a@sha256:0799d81892e65743ea606b4151ae3d13b29b70c0ac6f1636e67d3e8b79541150
cat /etc/moby
