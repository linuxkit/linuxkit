FROM linuxkit/alpine:07f7d136e427dc68154cd5edbb2b9576f9ac5213 AS build

RUN \
  apk add \
  bash \
  gcc \
  git \
  go \
  libc-dev \
  libseccomp-dev \
  linux-headers \
  make \
  socat \
  && true
ENV GOPATH=/go PATH=$PATH:/go/bin

ENV CRI_CONTAINERD_URL https://github.com/kubernetes-incubator/cri-containerd.git
#ENV CRI_CONTAINERD_BRANCH pull/NNN/head
ENV CRI_CONTAINERD_COMMIT ac8b0979fa634703e0a8d03df03eb51774fcff3d
RUN mkdir -p $GOPATH/src/github.com/kubernetes-incubator && \
    cd $GOPATH/src/github.com/kubernetes-incubator && \
    git clone $CRI_CONTAINERD_URL cri-containerd
WORKDIR $GOPATH/src/github.com/kubernetes-incubator/cri-containerd
RUN set -e; \
    if [ -n "$CRI_CONTAINERD_BRANCH" ] ; then \
        git fetch origin "$CRI_CONTAINERD_BRANCH"; \
    fi; \
    git checkout $CRI_CONTAINERD_COMMIT
RUN make static-binaries BUILD_TAGS="seccomp"

RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
# util-linux because a full ns-enter is required.
# example commands: /usr/bin/nsenter --net= -F -- <ip commandline>
#                   /usr/bin/nsenter --net=/var/run/netns/cni-5e8acebe-810d-c1b9-ced0-47be2f312fa8 -F -- <ip commandline>
# NB the first ("--net=") is actually not valid -- see https://github.com/kubernetes-incubator/cri-containerd/issues/245
RUN apk add --no-cache --initdb -p /out \
    alpine-baselayout \
    busybox \
    ca-certificates \
    iptables \
    util-linux \
    && true
# Remove apk residuals. We have a read-only rootfs, so apk is of no use.
RUN rm -rf /out/etc/apk /out/lib/apk /out/var/cache

RUN make DESTDIR=/out install

FROM scratch
WORKDIR /
ENTRYPOINT ["cri-containerd", "-v", "2", "--alsologtostderr", "--network-bin-dir", "/var/lib/cni/opt/bin", "--network-conf-dir", "/var/lib/cni/etc/net.d"]
COPY --from=build /out /
LABEL org.mobyproject.config='{"binds": ["/etc/resolv.conf:/etc/resolv.conf", "/run:/run:rshared,rbind", "/dev:/dev", "/tmp:/tmp", "/var:/var:rshared,rbind", "/var/lib/kubeadm:/etc/kubernetes", "/var/lib/cni/etc:/etc/cni:rshared,rbind", "/var/lib/cni/opt:/opt/cni:rshared,rbind", "/run/containerd/containerd.sock:/run/containerd/containerd.sock", "/var/lib/kubelet-plugins:/usr/libexec/kubernetes/kubelet-plugins:rshared,rbind"], "mounts": [{"type": "cgroup", "options": ["rw","nosuid","noexec","nodev","relatime"]}], "capabilities": ["all"], "rootfsPropagation": "shared", "pid": "host", "runtime": {"mkdir": ["/var/lib/kubeadm", "/var/lib/cni/etc/net.d", "/var/lib/cni/opt", "/var/lib/kubelet-plugins"]}}'
