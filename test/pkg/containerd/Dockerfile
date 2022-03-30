FROM linuxkit/containerd-dev:e6a8da1e270da1601ed1bb85bb44c4442e5d51be as containerd-dev
FROM linuxkit/alpine:33063834cf72d563cd8703467836aaa2f2b5a300 AS mirror
RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
# btrfs-progfs is required for btrfs test (mkfs.btrfs)
# util-linux is required for btrfs test (losetup)
# xfsprogs is required for xfs test (mkfs.xfs)
RUN apk add --no-cache --initdb -p /out \
    alpine-baselayout \
    busybox \
    btrfs-progs \
    btrfs-progs-dev \
    gcc \
    git \
    go \
    libc-dev \
    linux-headers \
    make \
    musl \
    util-linux \
    xfsprogs \
    tzdata
RUN rm -rf /out/etc/apk /out/lib/apk /out/var/cache
RUN cp /out/usr/share/zoneinfo/UTC /out/etc/localtime

FROM scratch
COPY --from=mirror /out/ /
COPY --from=containerd-dev /go/src/github.com/containerd/containerd /go/src/github.com/containerd/containerd/

# containerd 1.4.x does not support go modules; remove GO111MODULE=off when we switch to 1.5.x
ENV GOPATH=/go GO111MODULE=off
WORKDIR $GOPATH/src/github.com/containerd/containerd
ADD run.sh ./run.sh

ENTRYPOINT ["/bin/sh", "run.sh"]
