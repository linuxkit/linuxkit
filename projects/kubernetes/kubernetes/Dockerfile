FROM linuxkit/alpine:07f7d136e427dc68154cd5edbb2b9576f9ac5213 AS build

ENV kubernetes_version v1.8.1
ENV cni_version        v0.6.0

RUN apk add -U --no-cache \
  bash \
  coreutils \
  curl \
  findutils \
  git \
  go \
  grep \
  libc-dev \
  linux-headers \
  make \
  rsync \
  && true

ENV GOPATH=/go PATH=$PATH:/go/bin

ENV KUBERNETES_URL https://github.com/kubernetes/kubernetes.git
#ENV KUBERNETES_BRANCH pull/NNN/head
ENV KUBERNETES_COMMIT ${kubernetes_version}
RUN mkdir -p $GOPATH/src/github.com/kubernetes && \
    cd $GOPATH/src/github.com/kubernetes && \
    git clone $KUBERNETES_URL kubernetes
WORKDIR $GOPATH/src/github.com/kubernetes/kubernetes
RUN set -e; \
    if [ -n "$KUBERNETES_BRANCH" ] ; then \
        git fetch origin "$KUBERNETES_BRANCH"; \
    fi; \
    git checkout $KUBERNETES_COMMIT

RUN make WHAT="cmd/kubelet cmd/kubectl cmd/kubeadm"

RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
#coreutils needed for du -B for disk image checks made by kubelet
# example: $ du -s -B 1 /var/lib/kubelet/pods/...
#          du: unrecognized option: B
RUN apk add --no-cache --initdb -p /out \
    alpine-baselayout \
    busybox \
    ca-certificates \
    coreutils \
    curl \
    ebtables \
    ethtool \
    iproute2 \
    iptables \
    libc6-compat \
    musl \
    openssl \
    socat \
    util-linux \
    && true

RUN cp _output/bin/kubelet /out/usr/bin/kubelet
RUN cp _output/bin/kubeadm /out/usr/bin/kubeadm
RUN cp _output/bin/kubectl /out/usr/bin/kubectl

# Remove apk residuals. We have a read-only rootfs, so apk is of no use.
RUN rm -rf /out/etc/apk /out/lib/apk /out/var/cache

RUN rmdir /out/var/run && ln -nfs /run /out/var/run

RUN curl -fSL -o /out/root/cni.tgz https://github.com/containernetworking/plugins/releases/download/${cni_version}/cni-plugins-amd64-${cni_version}.tgz

ADD kubelet.sh /out/usr/bin/kubelet.sh
ADD kubeadm-init.sh /kubeadm-init.sh
RUN sed -e "s/@KUBERNETES_VERSION@/${kubernetes_version}/g" </kubeadm-init.sh >/out/usr/bin/kubeadm-init.sh && chmod +x /out/usr/bin/kubeadm-init.sh

FROM scratch
WORKDIR /
ENTRYPOINT ["/usr/bin/kubelet.sh"]
COPY --from=build /out /
ENV KUBECONFIG "/etc/kubernetes/admin.conf"
LABEL org.mobyproject.config='{"binds": ["/dev:/dev", "/etc/resolv.conf:/etc/resolv.conf", "/etc/os-release:/etc/os-release", "/run:/run:rshared,rbind", "/var:/var:rshared,rbind", "/var/lib/kubeadm:/etc/kubernetes", "/etc/kubelet.sh.conf:/etc/kubelet.sh.conf", "/etc/kubeadm:/etc/kubeadm", "/var/lib/kubelet-plugins:/usr/libexec/kubernetes/kubelet-plugins:rshared,rbind"], "mounts": [{"type": "cgroup", "options": ["rw","nosuid","noexec","nodev","relatime"]}], "capabilities": ["all"], "rootfsPropagation": "shared", "pid": "host", "runtime": {"mkdir": ["/var/lib/kubeadm", "/var/lib/cni/etc", "/var/lib/cni/opt", "/var/lib/kubelet-plugins"], "mounts": [{"type": "bind", "source": "/var/lib/cni/opt", "destination": "/opt/cni", "options": ["rw", "bind"]}, {"type": "bind", "source": "/var/lib/cni/etc", "destination": "/etc/cni", "options": ["rw", "bind"]}]}}'
