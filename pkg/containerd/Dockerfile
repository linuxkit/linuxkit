FROM linuxkit/alpine:12f0af976c42d4125a3ff3f5e636c95c0a7caa2a as alpine
RUN apk add tzdata binutils

WORKDIR $GOPATH/src/github.com/containerd/containerd
RUN cp bin/containerd bin/ctr bin/containerd-shim bin/containerd-shim-runc-v2 /usr/bin/

RUN strip /usr/bin/containerd /usr/bin/ctr /usr/bin/containerd-shim /usr/bin/containerd-shim-runc-v2

RUN mkdir -p /etc/init.d && ln -s /usr/bin/service /etc/init.d/020-containerd

WORKDIR /
COPY . .

FROM scratch
ENTRYPOINT []
WORKDIR /
COPY --from=alpine /usr/bin/containerd /usr/bin/ctr /usr/bin/containerd-shim /usr/bin/containerd-shim-runc-v2 /usr/bin/
COPY --from=alpine /etc/containerd/config.toml /etc/containerd/
COPY --from=alpine /usr/share/zoneinfo/UTC /etc/localtime
COPY --from=alpine /etc/init.d/ /etc/init.d/
COPY etc etc/
