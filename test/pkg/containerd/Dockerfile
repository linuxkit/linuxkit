FROM linuxkit/containerd-dev:0de800928578cd1915c68ed4834175df5b641db1 as containerd-dev
FROM linuxkit/alpine:35b33c6b03c40e51046c3b053dd131a68a26c37a AS mirror
RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
# btrfs-progfs is required for btrfs test (mkfs.btrfs)
# util-linux is required for btrfs test (losetup)
# xfsprogs is required for xfs test (mkfs.xfs)
RUN apk add --no-cache --initdb -p /out \
    alpine-baselayout \
    busybox \
    btrfs-progs \
    btrfs-progs-dev \
    device-mapper \
    e2fsprogs \    
    gcc \
    git \
    go \
    libc-dev \
    linux-headers \
    make \
    musl \
    util-linux \
    util-linux-misc \
    xfsprogs \
    tzdata

RUN rm -rf /out/etc/apk /out/lib/apk /out/var/cache
RUN cp /out/usr/share/zoneinfo/UTC /out/etc/localtime

FROM scratch
COPY --from=mirror /out/ /
COPY --from=containerd-dev /go/src/github.com/containerd/containerd /go/src/github.com/containerd/containerd/

RUN mv /sbin/dmsetup /sbin/dmsetup.orig
COPY dmsetup.sh /sbin/dmsetup
ENV GOPATH=/go
WORKDIR $GOPATH/src/github.com/containerd/containerd
ADD run.sh ./run.sh

ENTRYPOINT ["/bin/sh", "run.sh"]
