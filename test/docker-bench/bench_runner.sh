#!/bin/sh

echo "waiting for docker socket to be available..."

# wait for the docker runc container
while [ ! -e /var/run/docker.sock ]; do sleep 1; done

echo "found docker socket, starting docker bench..."

docker run -i --net host --pid host --cap-add audit_control -v /var/lib:/var/lib -v /var/run/docker.sock:/var/run/docker.sock --label docker_bench_security docker/docker-bench-security
