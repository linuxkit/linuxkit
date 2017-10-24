FROM weaveworks/weave:2.0.1@sha256:2d70caac7db33365482cc923d40ff8d3ec1238ae7fe06a00b3dde310d09f226e AS weave

# Nothing to do in here, just for COPY --from=weave below

FROM linuxkit/alpine:07f7d136e427dc68154cd5edbb2b9576f9ac5213 AS build

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

# https://github.com/ijc/swarmkit/tree/containerd-wip
ENV SWARMKIT_REPO=https://github.com/ijc/swarmkit
ENV SWARMKIT_BRANCH=containerd-wip
ENV SWARMKIT_COMMIT=8a09c038f1ba8f227a28b7f48ccc92a04edb85f5

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
