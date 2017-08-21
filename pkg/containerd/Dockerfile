FROM linuxkit/alpine:a120ad6aead3fe583eaa20e9b75a05ac1b3487da as alpine
RUN \
  apk add \
  btrfs-progs-dev \
  gcc \
  git \
  go \
  libc-dev \
  linux-headers \
  make \
  tzdata \
  && true

WORKDIR $GOPATH/src/github.com/containerd/containerd
RUN make binaries EXTRA_FLAGS="-buildmode pie" EXTRA_LDFLAGS="-extldflags \\\"-fno-PIC -static\\\""
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
