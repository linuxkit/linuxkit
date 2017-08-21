FROM linuxkit/alpine:a120ad6aead3fe583eaa20e9b75a05ac1b3487da AS build

ENV kubernetes_version v1.7.4
ENV weave_version      v2.0.1
ENV cni_version        v0.5.2

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

RUN rmdir /out/var/run && ln -nfs /run /out/var/run

RUN curl -fSL -o /tmp/cni.tgz https://github.com/containernetworking/cni/releases/download/v0.5.2/cni-amd64-${cni_version}.tgz && \
    mkdir -p /out/opt/cni/bin /out/etc/cni/net.d && \
    tar -xzf /tmp/cni.tgz -C /out/opt/cni/bin
RUN curl -fSL -o /out/etc/weave.yaml https://cloud.weave.works/k8s/v1.7/net?v=${weave_version} 
RUN curl -fSL -o /out/usr/bin/kubelet https://dl.k8s.io/${kubernetes_version}/bin/linux/amd64/kubelet && chmod 0755 /out/usr/bin/kubelet
RUN curl -fSL -o /out/usr/bin/kubeadm https://dl.k8s.io/${kubernetes_version}/bin/linux/amd64/kubeadm && chmod 0755 /out/usr/bin/kubeadm
RUN curl -fSL -o /out/usr/bin/kubectl https://dl.k8s.io/${kubernetes_version}/bin/linux/amd64/kubectl && chmod 0755 /out/usr/bin/kubectl

ADD kubelet.sh /out/usr/bin/kubelet.sh
ADD kubeadm-init.sh /kubeadm-init.sh
RUN sed -e "s/@KUBERNETES_VERSION@/${kubernetes_version}/g" </kubeadm-init.sh >/out/usr/bin/kubeadm-init.sh && chmod +x /out/usr/bin/kubeadm-init.sh

FROM scratch
WORKDIR /
ENTRYPOINT ["/usr/bin/kubelet.sh"]
COPY --from=build /out /
ENV KUBECONFIG "/etc/kubernetes/admin.conf"
LABEL org.mobyproject.config='{"binds": ["/dev:/dev", "/etc/resolv.conf:/etc/resolv.conf", "/run:/run", "/var:/var:rshared,rbind", "/var/lib/kubeadm:/etc/kubernetes", "/var/lib/cni/etc:/rootfs/etc/cni:rshared,rbind", "/var/lib/cni/opt:/rootfs/opt/cni:rshared,rbind"], "mounts": [{"type": "cgroup", "options": ["rw","nosuid","noexec","nodev","relatime"]}], "capabilities": ["all"], "rootfsPropagation": "shared", "pid": "host", "runtime": {"mkdir": ["/var/lib/kubeadm", "/var/lib/cni/etc", "/var/lib/cni/opt"]}}'
