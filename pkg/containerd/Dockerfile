FROM linuxkit/alpine:24fe1b6bbf1fc95b484741587acb10da6dbc9211 as alpine
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
ENV GOPATH=/go PATH=$PATH:/go/bin
RUN mkdir -p $GOPATH/src/github.com/containerd && \
  cd $GOPATH/src/github.com/containerd && \
  git clone https://github.com/containerd/containerd.git
WORKDIR $GOPATH/src/github.com/containerd/containerd
# CONTAINERD_COMMIT is defined in linuxkit/alpine
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
