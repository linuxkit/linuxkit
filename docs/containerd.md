## Using the system containerd

Here is a simple example script that will run a container using the system containerd.

You should run it from `/var` as the root filesystem is in RAM, and will use up memory.

```bash
#!/bin/sh

NAME=nginx
VERSION=latest

docker pull ${NAME}:${VERSION}
CONTAINER=$(docker create --net=host --security-opt apparmor=unconfined --cap-drop all --cap-add net_bind_service --oom-score-adj=-500 -v /var/log/nginx:/var/log/nginx -v /var/cache/nginx:/var/cache/nginx -v /var/run:/var/run ${NAME}:${VERSION})
docker run -v ${PWD}:/conf -v /var/run/docker.sock:/var/run/docker.sock --rm jess/riddler -f -bundle /conf ${CONTAINER}
rm -rf rootfs && mkdir rootfs
docker export ${CONTAINER} | tar -C rootfs -xf -
docker rm ${CONTAINER}

mkdir -p /var/log/nginx /var/cache/nginx

containerd-ctr containers start ${NAME} .
containerd-ctr containers
```

For debugging it helps to run `containerd-ctr containers start --attach ${NAME} .` It may
well turn out that you need to create directories that are empty volumes in docker.

For production, you will want to create the `config.json` offline and bundle it in with your
intii script, but you can create the rootfs online.
