FROM linuxkit/alpine:24fe1b6bbf1fc95b484741587acb10da6dbc9211 AS mirror
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
    util-linux
RUN rm -rf /out/etc/apk /out/lib/apk /out/var/cache

FROM scratch
COPY --from=mirror /out/ /
ENV GOPATH=/go
RUN mkdir -p $GOPATH/src/github.com/containerd && \
  cd $GOPATH/src/github.com/containerd && \
  git clone https://github.com/containerd/containerd.git
WORKDIR $GOPATH/src/github.com/containerd/containerd
# CONTAINERD_COMMIT is defined in linuxkit/alpine
RUN git checkout $CONTAINERD_COMMIT
ADD run.sh ./run.sh

ENTRYPOINT ["/bin/sh", "run.sh"]
LABEL org.mobyproject.config='{"net": "host", "capabilities": ["all"], "tmpfs": ["/tmp:exec"], "binds": ["/dev:/dev"]}'
