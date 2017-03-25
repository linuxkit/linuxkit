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
ENV CONTAINERD_COMMIT=8353da59c6ae7e1933aac2228df23541ef8b163f
RUN mkdir -p $GOPATH/src/github.com/docker && \
  cd $GOPATH/src/github.com/docker && \
  git clone https://github.com/docker/containerd.git
WORKDIR $GOPATH/src/github.com/docker/containerd
RUN git checkout $CONTAINERD_COMMIT
RUN make binaries GO_GCFLAGS="-buildmode pie --ldflags '-extldflags \"-fno-PIC -static\"'"
RUN cp bin/containerd bin/ctr bin/containerd-shim bin/dist /usr/bin/
WORKDIR /
COPY . .
