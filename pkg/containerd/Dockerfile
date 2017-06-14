FROM linuxkit/alpine:7cf5393e04fe0e26e9fa8f507379cacb64595918 as alpine
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
ENV CONTAINERD_COMMIT=c215531a8f63a98a69134e804fea4ee6d354bb90
RUN mkdir -p $GOPATH/src/github.com/containerd && \
  cd $GOPATH/src/github.com/containerd && \
  git clone https://github.com/containerd/containerd.git
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
