FROM linuxkit/alpine:f0af677747790eda63ee5e9b37b9002b965f48dc as alpine
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
