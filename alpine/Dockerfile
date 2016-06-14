FROM alpine:3.4

RUN \
  addgroup -g 50 docker && \
  adduser -G docker -u 1001 -s /bin/sh -D -g "Docker" docker && \
  passwd -d root && \
  apk update && apk upgrade && \
  apk add \
  e2fsprogs \
  chrony \
  git \
  xz \
  iptables \
  sfdisk \
  lvm2 \
  syslinux \
  openrc \
  busybox-initscripts \
  alpine-conf \
  bind-tools \
  openssh-client \
  strace \
  fuse \
  util-linux \
  cifs-utils \
  e2fsprogs-extra \
  openssl \
  && rm -rf /var/cache/apk/*

COPY etc /etc/
RUN mkdir -p /etc/docker

ADD kernel/aufs-utils.tar /
COPY mkinitrd.sh /bin/
COPY kernel/kernel-source-info /etc/
ADD kernel/kernel-patches.tar /etc/kernel-patches

COPY packages/proxy/slirp-proxy /usr/bin/
COPY packages/proxy/slirp-proxy /sbin/proxy-vsockd
COPY packages/proxy/etc /etc/
COPY packages/transfused/transfused /sbin/
COPY packages/transfused/etc /etc/
COPY packages/tap-vsockd/tap-vsockd /sbin/
COPY packages/tap-vsockd/etc /etc/
COPY packages/llmnrd/llmnrd /sbin/
COPY packages/llmnrd/etc /etc/
COPY packages/llmnrd/llmnrd.tar.gz /usr/share/src/
COPY packages/docker/bin/* /usr/bin/
COPY packages/docker/etc /etc/
COPY packages/diagnostics/diagnostics /usr/bin/
COPY packages/diagnostics/diagnostics-server /usr/bin/
COPY packages/diagnostics/etc /etc/
COPY packages/automount/etc /etc/
COPY packages/binfmt_misc/etc /etc/
COPY packages/dnsfix/etc /etc/
COPY packages/hostsettings/etc /etc/
COPY packages/hvtools/hv_kvp_daemon /sbin/
COPY packages/hvtools/hv_vss_daemon /sbin/
COPY packages/hvtools/etc /etc/
COPY packages/hvtools/hv_get_dhcp_info /sbin/
COPY packages/hvtools/hv_get_dns_info /sbin/
COPY packages/hvtools/hv_set_ifconfig /sbin/
COPY packages/hvtools/hvtools.tar.gz /usr/share/src/
COPY packages/chronyd/etc /etc/
COPY packages/userns/etc /etc/
COPY packages/userns/groupadd /usr/sbin
COPY packages/userns/useradd /usr/sbin
#COPY packages/nc-vsock/nc-vsock /usr/bin
COPY packages/vsudd/vsudd /sbin
COPY packages/vsudd/etc /etc
COPY packages/mobyconfig/mobyconfig /usr/bin
COPY packages/bootflag/bootflag /usr/bin
COPY packages/gummiboot/gummiboot.tar.gz /usr/share/src/
COPY packages/oom/etc /etc
COPY packages/9pmount-vsock/9pmount-vsock /sbin

RUN \
  rc-update add swap boot && \
  rc-update add sysctl boot && \
  rc-update add bootmisc boot && \
  rc-update add urandom boot && \
  rc-update add hostname boot && \
  rc-update add syslog boot && \
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
  rc-update add fsck && \
  rc-update add root && \
  rc-update add localmount && \
  rc-update add docker default && \
  rc-update add proxy default && \
  rc-update add transfused default && \
  rc-update add llmnrd default && \
  rc-update add automount boot && \
  rc-update add diagnostics default && \
  rc-update add binfmt_misc default && \
  rc-update add dnsfix boot && \
  rc-update add hostsettings boot && \
  rc-update add hv_kvp_daemon default && \
  rc-update add hv_vss_daemon default && \
  rc-update add vsudd default && \
  rc-update add oom default && \
  true

# we do not need to restart syslog, as probably not running
RUN \
  sed -i 's@/etc/init.d/syslog --quiet restart@@' /sbin/setup-disk

COPY init /

CMD ["/bin/sh"]
