# FIXME: maintainers please push and update linuxkit/alpine
FROM linuxkit/alpine
# btrfs-progfs is required for btrfs test (mkfs.btrfs)
# util-linux is required for btrfs test (losetup)
RUN \
  apk add \
  btrfs-progs \
  btrfs-progs-dev \
  gcc \
  git \
  go \
  libc-dev \
  linux-headers \
  make \
  util-linux
ENV GOPATH=/root/go
RUN mkdir -p $GOPATH/src/github.com/containerd && \
  cd $GOPATH/src/github.com/containerd && \
  git clone https://github.com/containerd/containerd.git
WORKDIR $GOPATH/src/github.com/containerd/containerd
# CONTAINERD_COMMIT is defined in linuxkit/alpine
RUN git checkout $CONTAINERD_COMMIT

# unset -race (does not work on alpine; see golang/go#14481)
ENV TESTFLAGS=
ENTRYPOINT ["make", "root-test"]
LABEL org.mobyproject.config='{"net": "host", "capabilities": ["all"], "tmpfs": ["/tmp:exec"], "binds": ["/dev:/dev"]}'
