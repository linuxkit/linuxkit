FROM linuxkit/alpine:630ee558e4869672fae230c78364e367b8ea67a9 AS build

RUN \
  apk update && apk upgrade && \
  apk add --no-cache \
  ca-certificates \
  gcc \
  git \
  go \
  libc-dev \
  make \
  && true

ENV GOPATH=/root/go

# https://github.com/ijc25/swarmkit/tree/containerd-wip
ENV SWARMKIT_REPO=https://github.com/ijc25/swarmkit
ENV SWARMKIT_BRANCH=containerd-wip
ENV SWARMKIT_COMMIT=cb429323fa86df0c279a8d629e674f79a20e01e3

RUN mkdir -p $GOPATH/src/github.com/docker && \
  cd $GOPATH/src/github.com/docker && \
  git clone $SWARMKIT_REPO
WORKDIR $GOPATH/src/github.com/docker/swarmkit
RUN [ -z "$SWARMKIT_BRANCH" ] || git fetch origin $SWARMKIT_BRANCH
RUN git checkout $SWARMKIT_COMMIT

RUN make binaries GO_GCFLAGS="-buildmode pie --ldflags '-extldflags \"-fno-PIC -static\"'"

RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
RUN apk add --no-cache --initdb -p /out \
    alpine-baselayout \
    busybox \
    ca-certificates \
    iptables \
    musl \
    util-linux

RUN mkdir -p /out/usr/bin/ /out/etc
RUN cp bin/swarmd bin/swarmctl /out/usr/bin/
RUN strip /out/usr/bin/swarmd /out/usr/bin/swarmctl

FROM scratch
WORKDIR /
ENTRYPOINT []
COPY --from=build /out /
CMD ["/usr/bin/swarmd", "--containerd-addr=/run/containerd/containerd.sock", "--log-level=debug", "--state-dir=/var/lib/swarmd"]
