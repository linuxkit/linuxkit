FROM linuxkit/alpine:8bb8664eec04e02a8a131c53aa7d5d94119270ef as alpine
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
ENV GOPATH=/go PATH=$PATH:/go/bin
# CONTAINERD_REPO and CONTAINERD_COMMIT are defined in linuxkit/alpine
RUN mkdir -p $GOPATH/src/github.com/containerd && \
  cd $GOPATH/src/github.com/containerd && \
  git clone $CONTAINERD_REPO
WORKDIR $GOPATH/src/github.com/containerd/containerd
RUN git checkout $CONTAINERD_COMMIT
RUN make binaries EXTRA_FLAGS="-buildmode pie" EXTRA_LDFLAGS="-extldflags \\\"-fno-PIC -static\\\""
RUN cp bin/containerd bin/ctr bin/containerd-shim bin/dist /usr/bin/

ADD cmd /go/src/cmd
RUN cd /go/src/cmd/service && ./skanky-vendor.sh $GOPATH/src/github.com/containerd/containerd
RUN go-compile.sh /go/src/cmd/service

WORKDIR /
COPY . .

FROM scratch
ENTRYPOINT []
WORKDIR /
COPY --from=alpine /usr/bin/containerd /usr/bin/ctr /usr/bin/dist /usr/bin/containerd-shim /go/bin/service /usr/bin/
COPY --from=alpine /etc/containerd/config.toml /etc/containerd/
COPY --from=alpine /usr/share/zoneinfo/UTC /etc/localtime
COPY etc etc/
