FROM linuxkit/alpine:316c3f9d85c21fdd8bc7479e81d290f85bf60eb0 as alpine

RUN apk add tzdata binutils
RUN mkdir -p /etc/init.d && ln -s /usr/bin/service /etc/init.d/020-containerd

FROM linuxkit/containerd-dev:ee5755eb38c0a6b6aef2658b757652d82556c30c as containerd-dev

FROM scratch
ENTRYPOINT []
WORKDIR /
COPY --from=containerd-dev /usr/bin/containerd /usr/bin/ctr /usr/bin/containerd-shim /usr/bin/containerd-shim-runc-v2 /usr/bin/
COPY --from=alpine /usr/share/zoneinfo/UTC /etc/localtime
COPY --from=alpine /etc/init.d/ /etc/init.d/
COPY etc etc/
COPY --from=alpine /etc/apk /etc/apk/
COPY --from=alpine /lib/apk /lib/apk/
