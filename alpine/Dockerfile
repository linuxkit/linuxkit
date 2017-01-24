# Tag: 5837a236153f00bb215642e3e0639252eb49cdf9
FROM mobylinux/alpine-base@sha256:f6f12aebe2af07c9250014ff283485dbdf082bd9cfbd74aad27a3d2dcf13e0b1

ENV ARCH=x86_64

RUN \
  addgroup -g 50 docker && \
  adduser -G docker -u 1001 -s /bin/sh -D -g "Docker" docker && \
  passwd -d root && \
  adduser -D -H -s /sbin/nologin dockremap

COPY . .
RUN rm Dockerfile

RUN \
  rc-update add sysctl boot && \
  rc-update add bootmisc boot && \
  rc-update add urandom boot && \
  rc-update add hostname boot && \
  rc-update add vsudd boot && \
  rc-update add sysklogd boot && \
  rc-update add hwclock boot && \
  rc-update add tap-vsockd boot && \
  rc-update add networking boot && \
  rc-update add dhcpcd boot && \
  rc-update add acpid default && \
  rc-update add chronyd default && \
  rc-update add savecache shutdown && \
  rc-update add killprocs shutdown && \
  rc-update add mount-ro shutdown && \
  rc-update add dmesg sysinit && \
  rc-update add devfs sysinit && \
  rc-update add hwdrivers sysinit && \
  rc-update add sysfs && \
  rc-update add procfs && \
  rc-update add sysfsconf && \
  rc-update add fsck && \
  rc-update add crond && \
  rc-update add local && \
  rc-update add localmount && \
  rc-update add docker default && \
  rc-update add proxy default && \
  rc-update add transfused default && \
  rc-update add automount sysinit && \
  rc-update add diagnostics default && \
  rc-update add database-mac boot && \
  rc-update add database-windows boot && \
  rc-update add database-aws default && \
  rc-update add database-gcp default && \
  rc-update add hostsettings default && \
  rc-update add windowsnet boot && \
  rc-update add hv_kvp_daemon default && \
  rc-update add hv_vss_daemon default && \
  rc-update add oom default && \
  rc-update add test default && \
  rc-update add containerd default && \
  rc-update add aws default && \
  rc-update add azure default && \
  rc-update add gcp-hostname default && \
  rc-update add gcp-startup default && \
  true

CMD ["/mkinitrd.sh"]
