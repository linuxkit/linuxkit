#FROM linuxkit/alpine:9bcf61f605ef0ce36cc94d59b8eac307862de6e1 AS build
# XXX needs ebtables ethtool iproute2 libc6-compat socat
FROM alpine:3.6 AS build

ENV kubernetes_version v1.6.1
ENV weave_version      v1.9.4
ENV cni_version        0799f5732f2a11b329d9e3d51b9c8f2e3759f2ff

ENV kube_release_artefacts "https://dl.k8s.io/${kubernetes_version}/bin/linux/amd64"

RUN apk add -U --no-cache \
  curl \
  && true

RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
RUN apk add --no-cache --initdb -p /out \
    alpine-baselayout \
    busybox \
    ca-certificates \
    curl \
    ebtables \
    ethtool \
    iproute2 \
    iptables \
    libc6-compat \
    musl \
    socat \
    util-linux \
    && true
# Remove apk residuals. We have a read-only rootfs, so apk is of no use.
RUN rm -rf /out/etc/apk /out/lib/apk /out/var/cache

RUN curl -fSL -o /tmp/cni.tgz https://dl.k8s.io/network-plugins/cni-amd64-${cni_version}.tar.gz && \
    mkdir -p /out/opt/cni /out/etc/cni/net.d && \
    tar -xzf /tmp/cni.tgz -C /out/opt/cni
RUN curl -fSL -o /out/etc/weave.yaml https://cloud.weave.works/k8s/v1.6/net?v=${weave_version} 
RUN curl -fSL -o /out/usr/bin/kubelet https://dl.k8s.io/${kubernetes_version}/bin/linux/amd64/kubelet && chmod 0755 /out/usr/bin/kubelet
RUN curl -fSL -o /out/usr/bin/kubeadm https://dl.k8s.io/${kubernetes_version}/bin/linux/amd64/kubeadm && chmod 0755 /out/usr/bin/kubeadm
RUN curl -fSL -o /out/usr/bin/kubectl https://dl.k8s.io/${kubernetes_version}/bin/linux/amd64/kubectl && chmod 0755 /out/usr/bin/kubectl

ADD kubelet.sh /out/usr/bin/kubelet.sh
ADD kubeadm-init.sh /out/usr/bin/kubeadm-init.sh

FROM scratch
WORKDIR /
ENTRYPOINT ["/usr/bin/kubelet.sh"]
COPY --from=build /out /
ENV KUBECONFIG "/etc/kubernetes/admin.conf"
LABEL org.mobyproject.config='{"binds": ["/dev:/dev", "/var:/var:rshared,rbind", "/var/lib/kubeadm:/etc/kubernetes", "/etc/cni:/rootfs/etc/cni:rshared,rbind", "/opt/cni:/rootfs/opt/cni:rshared,rbind"], "mounts": [{"type": "cgroup", "options": ["rw","nosuid","noexec","nodev","relatime"]}], "capabilities": ["all"], "rootfsPropagation": "shared", "pid": "host"}'
