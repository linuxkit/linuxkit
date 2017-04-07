#! /bin/sh
docker build -t mobylinux/etcd .

docker build -t etcd.local -f Dockerfile.local .
