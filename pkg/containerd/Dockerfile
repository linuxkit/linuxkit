# Dockerfile to build linuxkit/containerd for linuxkit
FROM linuxkit/alpine:7f3944798557de5518a56e3437d7ed982701f224 as alpine

RUN apk add tzdata binutils
RUN mkdir -p /etc/init.d && ln -s /usr/bin/service /etc/init.d/020-containerd

FROM linuxkit/containerd-dev:405449f30ba384a0f67ad1b75f78a501604f2e44 as containerd-dev

FROM scratch
ENTRYPOINT []
WORKDIR /
COPY --from=containerd-dev /usr/bin/containerd /usr/bin/ctr /usr/bin/containerd-shim-runc-v2 /usr/bin/
COPY --from=alpine /usr/share/zoneinfo/UTC /etc/localtime
COPY --from=alpine /etc/init.d/ /etc/init.d/
COPY etc etc/
COPY --from=alpine /etc/apk /etc/apk/
COPY --from=alpine /lib/apk /lib/apk/
