FROM linuxkit/alpine:6ed3b299f5243acb6459b4993549c5045e4ad7f4 AS mirror
RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
# btrfs-progfs is required for btrfs test (mkfs.btrfs)
# util-linux is required for btrfs test (losetup)
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
LABEL org.mobyproject.config='{"capabilities": ["all"], "tmpfs": ["/tmp:exec"], "binds": ["/var:/var", "/etc/resolv.conf:/etc/resolv.conf", "/usr/bin/runc:/usr/bin/runc", "/usr/bin/containerd:/usr/bin/containerd", "/usr/bin/containerd-shim:/usr/bin/containerd-shim"], "mounts": [{"type": "cgroup", "options": ["rw","nosuid","noexec","nodev","relatime"]}],}'
