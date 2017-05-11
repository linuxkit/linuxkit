FROM weaveworks/weave@sha256:05172329b6ff72099db7bb891ac311b89948a3064ca9b8641c6b4abe38548677 AS weave

# Nothing to do in here, just for COPY --from=weave below

FROM linuxkit/alpine:9bcf61f605ef0ce36cc94d59b8eac307862de6e1 AS build

RUN \
  apk update && apk upgrade && \
  apk add --no-cache \
  bash \
  ca-certificates \
  gcc \
  git \
  go \
  libc-dev \
  linux-headers \
  make \
  && true

ENV GOPATH=/root/go

#####################################################################
# Output filesystem skeleton

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

RUN mkdir -p /out/usr/bin/ /out/etc  /out/opt/cni/bin /out/etc/cni/net.d

#####################################################################
# Swarmd

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

RUN cp bin/swarmd bin/swarmctl /out/usr/bin/
RUN strip /out/usr/bin/swarmd /out/usr/bin/swarmctl

#####################################################################
# CNI

ENV CNI_REPO=https://github.com/containernetworking/cni
ENV CNI_COMMIT=v0.5.2

RUN mkdir -p $GOPATH/src/github.com/containernetworking && \
  cd $GOPATH/src/github.com/containernetworking && \
  git clone $CNI_REPO
WORKDIR $GOPATH/src/github.com/containernetworking/cni
RUN [ -z "$CNI_BRANCH" ] || git fetch origin $CNI_BRANCH
RUN git checkout $CNI_COMMIT

RUN ./build.sh -buildmode pie --ldflags '-extldflags "-fno-PIC -static"'

RUN cp bin/bridge bin/host-local bin/dhcp /out/opt/cni/bin/

#####################################################################
# Weave

COPY --from=weave /usr/bin/weaveutil /out/opt/cni/bin/weave-net
RUN ln -s weave-net /out/opt/cni/bin/weave-ipam

FROM scratch
WORKDIR /
ENTRYPOINT []
COPY --from=build /out /
CMD ["/usr/bin/swarmd", "--containerd-addr=/run/containerd/containerd.sock", "--log-level=debug", "--state-dir=/var/lib/swarmd"]
