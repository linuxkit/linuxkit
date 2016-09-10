FROM mobylinux/alpine-base:77722e15bd14c8de88e2d2a8235bac75ee6187b3

ENV ARCH=x86_64

RUN \
  addgroup -g 50 docker && \
  adduser -G docker -u 1001 -s /bin/sh -D -g "Docker" docker && \
  passwd -d root

COPY etc /etc/
RUN mkdir -p /etc/docker

ADD kernel/$ARCH/aufs-utils.tar /
COPY mkinitrd.sh /bin/
COPY kernel/$ARCH/kernel-source-info /etc/
ADD kernel/$ARCH/kernel-patches.tar /etc/kernel-patches
ADD kernel/$ARCH/kernel-modules.tar /

COPY packages/proxy/usr /usr/
COPY packages/proxy/sbin /sbin/
COPY packages/proxy/etc /etc/
COPY packages/transfused/sbin /sbin/
COPY packages/transfused/etc /etc/
COPY packages/tap-vsockd/sbin /sbin/
COPY packages/tap-vsockd/etc /etc/
COPY packages/docker/usr /usr/
COPY packages/docker/etc /etc/
COPY packages/diagnostics/usr /usr/
COPY packages/diagnostics/etc /etc/
COPY packages/automount/etc /etc/
COPY packages/binfmt_misc/etc /etc/
COPY packages/hostsettings/etc /etc/
COPY packages/hvtools/sbin /sbin/
COPY packages/hvtools/etc /etc/
COPY packages/hvtools/usr /usr/
COPY packages/chronyd/etc /etc/
COPY packages/userns/etc /etc/
COPY packages/nc-vsock/usr /usr/
COPY packages/vsudd/sbin /sbin/
COPY packages/vsudd/etc /etc/
COPY packages/mobyconfig/usr /usr/
COPY packages/mobyplatform/usr /usr/
COPY packages/oom/etc /etc/
COPY packages/9pmount-vsock/sbin /sbin/
COPY packages/test/etc /etc/
COPY packages/test/usr /usr/
COPY packages/iptables/usr /usr/
COPY packages/containerd/etc /etc/
COPY packages/aws/etc /etc/
COPY packages/azure/etc /etc/

RUN \
  rc-update add swap boot && \
  rc-update add sysctl boot && \
  rc-update add bootmisc boot && \
  rc-update add urandom boot && \
  rc-update add hostname boot && \
  rc-update add vsudd boot && \
  rc-update add sysklogd boot && \
  rc-update add hwclock boot && \
  rc-update add networking boot && \
  rc-update add acpid default && \
  rc-update add chronyd default && \
  rc-update add savecache shutdown && \
  rc-update add killprocs shutdown && \
  rc-update add mount-ro shutdown && \
  rc-update add dmesg sysinit && \
  rc-update add devfs sysinit && \
  rc-update add hwdrivers sysinit && \
  rc-update add sysfs && \
  rc-update add sysfsconf && \
  rc-update add fsck && \
  rc-update add root && \
  rc-update add crond && \
  rc-update add localmount && \
  rc-update add docker default && \
  rc-update add proxy default && \
  rc-update add transfused default && \
  rc-update add automount boot && \
  rc-update add diagnostics default && \
  rc-update add binfmt_misc default && \
  rc-update add hostsettings boot && \
  rc-update add hv_kvp_daemon default && \
  rc-update add hv_vss_daemon default && \
  rc-update add oom default && \
  rc-update add test default && \
  rc-update add containerd default && \
  rc-update add aws default && \
  rc-update add azure default && \
  true

COPY init /

RUN adduser -D -H -s /sbin/nologin dockremap

RUN cd /usr/bin && \
  ln -s docker-runc runc && \
  ln -s docker-containerd-shim containerd-shim && \
  ln -s docker-containerd-ctr containerd-ctr && \
  ln -s docker-containerd containerd

CMD ["/bin/sh"]
