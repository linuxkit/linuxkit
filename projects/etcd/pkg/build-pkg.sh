#! /bin/sh

docker build -t moby/etcd -f Dockerfile.etcd .
docker build -t etcd.local -f Dockerfile.etcd.local .

docker build -t moby/prom-us-central1-f -f Dockerfile.prom.us-central1-f .
docker build -t moby/prom-local -f Dockerfile.prom.local .
