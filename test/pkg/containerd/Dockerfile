FROM linuxkit/alpine:85eafb6391ee71af83d4338df70b416b00859469 AS mirror
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

RUN apk add git
ENV GOPATH=/out/go
RUN mkdir -p $GOPATH/src/github.com/containerd && \
  cd $GOPATH/src/github.com/containerd && \
  git clone https://github.com/containerd/containerd.git
WORKDIR $GOPATH/src/github.com/containerd/containerd
# CONTAINERD_COMMIT is defined in linuxkit/alpine
RUN git checkout $CONTAINERD_COMMIT

FROM scratch
COPY --from=mirror /out/ /
ENV GOPATH=/go
WORKDIR $GOPATH/src/github.com/containerd/containerd
ADD run.sh ./run.sh

ENTRYPOINT ["/bin/sh", "run.sh"]
LABEL org.mobyproject.config='{"capabilities": ["all"], "tmpfs": ["/tmp"], "binds": ["/dev:/dev", "/var/lib:/var/lib", "/etc/resolv.conf:/etc/resolv.conf", "/usr/bin/runc:/usr/bin/runc", "/usr/bin/containerd:/usr/bin/containerd", "/usr/bin/containerd-shim:/usr/bin/containerd-shim"], "mounts": [{"type": "cgroup", "options": ["rw","nosuid","noexec","nodev","relatime"]}],}'
