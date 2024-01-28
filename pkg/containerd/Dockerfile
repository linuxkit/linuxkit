# Dockerfile to build linuxkit/containerd for linuxkit
FROM linuxkit/alpine:a720d2b1b83556653abd52d61faa0fe020c2b161 as alpine

RUN apk add tzdata binutils
RUN mkdir -p /etc/init.d && ln -s /usr/bin/service /etc/init.d/020-containerd

FROM linuxkit/containerd-dev:8f880c3f91e9d43d8d3e754847335e57980836ed as containerd-dev

FROM scratch
ENTRYPOINT []
WORKDIR /
COPY --from=containerd-dev /usr/bin/containerd /usr/bin/ctr /usr/bin/containerd-shim /usr/bin/containerd-shim-runc-v2 /usr/bin/
COPY --from=alpine /usr/share/zoneinfo/UTC /etc/localtime
COPY --from=alpine /etc/init.d/ /etc/init.d/
COPY etc etc/
COPY --from=alpine /etc/apk /etc/apk/
COPY --from=alpine /lib/apk /lib/apk/
