FROM linuxkit/alpine:02d7e748614512ea099e8c76254e41bcea03ba8f AS mirror
RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
# btrfs-progfs is required for btrfs test (mkfs.btrfs)
# util-linux is required for btrfs test (losetup)
# xfsprogs is required for xfs test (mkfs.xfs)
# containerd v1.1.x requires go1.10 which is currently a separate package in Alpine 3.7
RUN apk add --no-cache --initdb -p /out \
    alpine-baselayout \
    busybox \
    btrfs-progs \
    btrfs-progs-dev \
    gcc \
    git \
    go1.10 \
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
COPY --from=mirror /go/src/github.com/containerd/containerd /go/src/github.com/containerd/containerd/
ENV GOPATH=/go
WORKDIR $GOPATH/src/github.com/containerd/containerd
ADD run.sh ./run.sh

ENTRYPOINT ["/bin/sh", "run.sh"]
LABEL org.mobyproject.config='{"capabilities": ["all"], "tmpfs": ["/tmp"], "binds": ["/dev:/dev", "/var/lib:/var/lib", "/etc/resolv.conf:/etc/resolv.conf", "/usr/bin/runc:/usr/bin/runc", "/usr/bin/containerd:/usr/bin/containerd", "/usr/bin/containerd-shim:/usr/bin/containerd-shim"], "mounts": [{"type": "cgroup", "options": ["rw","nosuid","noexec","nodev","relatime"]}],}'
