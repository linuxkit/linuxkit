FROM linuxkit/alpine:9bcf61f605ef0ce36cc94d59b8eac307862de6e1 AS build

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
ENV SWARMKIT_COMMIT=4a484ccb498bee117fe6167d5a5e7ea0f6d4f2e9

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

# Remove apk residuals. We have a read-only rootfs, so apk is of no use.
RUN rm -rf /out/etc/apk /out/lib/apk /out/var/cache

RUN mkdir -p /out/usr/bin/ /out/etc
RUN cp bin/swarmd bin/swarmctl /out/usr/bin/
RUN strip /out/usr/bin/swarmd /out/usr/bin/swarmctl

FROM scratch
WORKDIR /
ENTRYPOINT []
COPY --from=build /out /
CMD ["/usr/bin/swarmd", "--containerd-addr=/run/containerd/containerd.sock", "--log-level=debug", "--state-dir=/var/lib/swarmd"]
