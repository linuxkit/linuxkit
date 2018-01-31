FROM linuxkit/alpine:4d6cfc5de9c29ad5cb633db9638660ba22d2fea5 as alpine
RUN apk add tzdata

WORKDIR $GOPATH/src/github.com/containerd/containerd
RUN cp bin/containerd bin/ctr bin/containerd-shim /usr/bin/

RUN mkdir -p /etc/init.d && ln -s /usr/bin/service /etc/init.d/020-containerd

WORKDIR /
COPY . .

FROM scratch
ENTRYPOINT []
WORKDIR /
COPY --from=alpine /usr/bin/containerd /usr/bin/ctr /usr/bin/containerd-shim /usr/bin/
COPY --from=alpine /etc/containerd/config.toml /etc/containerd/
COPY --from=alpine /usr/share/zoneinfo/UTC /etc/localtime
COPY --from=alpine /etc/init.d/ /etc/init.d/
COPY etc etc/
