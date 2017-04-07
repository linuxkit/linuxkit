FROM golang:1.7-alpine3.5
RUN \
  apk update && apk upgrade -a && \
  apk add --no-cache \
  btrfs-progs-dev \
  gcc \
  git \
  libc-dev \
  linux-headers \
  make \
  && true
ENV CONTAINERD_COMMIT=e5c8c5634a1fa82da41c1b707f8a9889bcfca248
RUN mkdir -p $GOPATH/src/github.com/containerd && \
  cd $GOPATH/src/github.com/containerd && \
  git clone https://github.com/containerd/containerd.git
WORKDIR $GOPATH/src/github.com/containerd/containerd
RUN git checkout $CONTAINERD_COMMIT
RUN make binaries GO_GCFLAGS="-buildmode pie --ldflags '-extldflags \"-fno-PIC -static\"'"
RUN cp bin/containerd bin/ctr bin/containerd-shim bin/dist /usr/bin/
WORKDIR /
RUN printf "FROM scratch\nCOPY /usr/bin/* /usr/bin/\n" > Dockerfile
CMD ["tar", "cf", "-", "Dockerfile", "usr/bin/containerd", "usr/bin/ctr", "usr/bin/containerd-shim", "usr/bin/dist"]
