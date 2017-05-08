FROM golang:1.8-alpine
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
ENV CONTAINERD_COMMIT=25a161bf5d4483bd0bea9e38b0e8fe3ecb17b53e
RUN mkdir -p $GOPATH/src/github.com/containerd && \
  cd $GOPATH/src/github.com/containerd && \
  git clone https://github.com/containerd/containerd.git
WORKDIR $GOPATH/src/github.com/containerd/containerd
RUN git checkout $CONTAINERD_COMMIT
RUN make binaries GO_GCFLAGS="-buildmode pie --ldflags '-extldflags \"-fno-PIC -static\"'"
RUN cp bin/containerd bin/ctr bin/containerd-shim bin/dist /usr/bin/
WORKDIR /
COPY . .
RUN printf "FROM scratch\nCOPY /usr/bin/* /usr/bin/\nCOPY /etc/containerd/config.toml /etc/containerd/\n" > Dockerfile
CMD ["tar", "cf", "-", "Dockerfile", "usr/bin/containerd", "usr/bin/ctr", "usr/bin/containerd-shim", "usr/bin/dist", "etc/containerd/config.toml"]
