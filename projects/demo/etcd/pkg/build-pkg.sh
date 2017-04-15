#! /bin/sh
docker build -t linuxkit/etcd .

docker build -t etcd.local -f Dockerfile.local .
