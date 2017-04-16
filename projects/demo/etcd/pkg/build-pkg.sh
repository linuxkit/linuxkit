#! /bin/sh
docker build -t moby/etcd .

docker build -t etcd.local -f Dockerfile.local .
