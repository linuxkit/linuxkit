FROM golang:1.7-alpine3.5
RUN \
  apk update && apk upgrade -a && \
  apk add --no-cache \
  gcc \
  git \
  libc-dev \
  make \
  && true
ENV CONTAINERD_COMMIT=3b79682548339895fcf9976f60ddea8abc5fc97e
RUN mkdir -p $GOPATH/src/github.com/docker && \
  cd $GOPATH/src/github.com/docker && \
  git clone https://github.com/docker/containerd.git
WORKDIR $GOPATH/src/github.com/docker/containerd
RUN git checkout $CONTAINERD_COMMIT
RUN make binaries GO_GCFLAGS="-buildmode pie --ldflags '-extldflags \"-fno-PIC -static\"'"
RUN cp bin/containerd bin/ctr bin/containerd-shim /usr/bin/
WORKDIR /
COPY . .
