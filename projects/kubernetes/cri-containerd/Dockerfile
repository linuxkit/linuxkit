FROM linuxkit/alpine:4a9e5f80a774bbea494d49324428a22c6f018865 AS build

RUN \
  apk add \
  bash \
  gcc \
  git \
  go \
  libc-dev \
  make \
  && true
ENV GOPATH=/go PATH=$PATH:/go/bin

ENV CRI_CONTAINERD_URL https://github.com/kubernetes-incubator/cri-containerd.git
#ENV CRI_CONTAINERD_BRANCH ...
ENV CRI_CONTAINERD_COMMIT 86a0f6a59b0d61381d60473680f1b6283fbfd8f9
RUN mkdir -p $GOPATH/src/github.com/kubernetes-incubator && \
    cd $GOPATH/src/github.com/kubernetes-incubator && \
    git clone $CRI_CONTAINERD_URL cri-containerd
WORKDIR $GOPATH/src/github.com/kubernetes-incubator/cri-containerd
RUN set -e; \
    if [ -n "$CRI_CONTAINERD_BRANCH" ] ; then \
        git fetch origin "$CRI_CONTAINERD_BRANCH"; \
    fi; \
    git checkout $CRI_CONTAINERD_COMMIT
RUN make static-binaries

RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
RUN apk add --no-cache --initdb -p /out \
    alpine-baselayout \
    busybox \
    ca-certificates \
    && true
# Remove apk residuals. We have a read-only rootfs, so apk is of no use.
RUN rm -rf /out/etc/apk /out/lib/apk /out/var/cache

RUN make DESTDIR=/out install

FROM scratch
WORKDIR /

ENTRYPOINT ["cri-containerd", "-v", "2", "--alsologtostderr"]
COPY --from=build /out /
LABEL org.mobyproject.config='{"binds": ["/etc/resolv.conf:/etc/resolv.conf", "/run:/run", "/tmp:/tmp", "/var:/var:rshared,rbind", "/var/lib/kubeadm:/etc/kubernetes", "/etc/cni:/etc/cni:rshared,rbind", "/opt/cni:/opt/cni:rshared,rbind", "/run/containerd/containerd.sock:/run/containerd/containerd.sock"], "mounts": [{"type": "cgroup", "options": ["rw","nosuid","noexec","nodev","relatime"]}], "capabilities": ["all"], "rootfsPropagation": "shared", "pid": "host"}'
