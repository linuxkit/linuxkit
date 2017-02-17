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
docker run mobylinux/check-kernel-config:3d64e3ddd9315bdc1e82ea652ea27c8b149be5d3@sha256:450c641e045b346e11f3e892d31d0bd9a94874e0129be4715d3741f252439140
cat /etc/moby
