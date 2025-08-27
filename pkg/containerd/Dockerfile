# Dockerfile to build linuxkit/containerd for linuxkit
FROM linuxkit/alpine:35b33c6b03c40e51046c3b053dd131a68a26c37a as alpine

RUN apk add tzdata binutils
RUN mkdir -p /etc/init.d && ln -s /usr/bin/service /etc/init.d/020-containerd

FROM linuxkit/containerd-dev:0de800928578cd1915c68ed4834175df5b641db1 as containerd-dev

FROM scratch
ENTRYPOINT []
WORKDIR /
COPY --from=containerd-dev /usr/bin/containerd /usr/bin/ctr /usr/bin/containerd-shim-runc-v2 /usr/bin/
COPY --from=alpine /usr/share/zoneinfo/UTC /etc/localtime
COPY --from=alpine /etc/init.d/ /etc/init.d/
COPY etc etc/
COPY --from=alpine /etc/apk /etc/apk/
COPY --from=alpine /lib/apk /lib/apk/
