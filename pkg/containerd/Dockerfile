FROM linuxkit/alpine:630ee558e4869672fae230c78364e367b8ea67a9 as alpine
RUN \
  apk add \
  btrfs-progs-dev \
  gcc \
  git \
  go \
  libc-dev \
  linux-headers \
  make \
  && true
ENV GOPATH=/root/go
ENV CONTAINERD_COMMIT=bdf9f5f7388e8203e63a74b89800f7f3dd4a7743
RUN mkdir -p $GOPATH/src/github.com/containerd && \
  cd $GOPATH/src/github.com/containerd && \
  git clone https://github.com/containerd/containerd.git
WORKDIR $GOPATH/src/github.com/containerd/containerd
RUN git checkout $CONTAINERD_COMMIT
RUN make binaries EXTRA_FLAGS="-buildmode pie" EXTRA_LDFLAGS="-extldflags \\\"-fno-PIC -static\\\""
RUN cp bin/containerd bin/ctr bin/containerd-shim bin/dist /usr/bin/
WORKDIR /
COPY . .

FROM scratch
ENTRYPOINT []
WORKDIR /
COPY --from=alpine /usr/bin/containerd /usr/bin/ctr /usr/bin/dist /usr/bin/containerd-shim /usr/bin/
COPY --from=alpine /etc/containerd/config.toml /etc/containerd/
