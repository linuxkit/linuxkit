# FROM alpine:3.3
FROM alpine@sha256:1849e75e25b5a005781b32e7ce0ec2892c85ef3d40d76861a6d3c721f1acc353

COPY repositories /etc/apk/

RUN \
  addgroup -g 50 docker && \
  adduser -G docker -u 1001 -s /bin/sh -D -g "Docker" docker && \
  passwd -d root && \
  apk update && apk upgrade && \
  apk add \
  e2fsprogs chrony \
  git xz iptables \
  sfdisk lvm2 syslinux \
  openrc busybox-initscripts \
  alpine-conf \
  bind-tools \
  openssh-client \
  strace fuse

COPY etc /etc/

ADD kernel/aufs-utils.tar /
COPY mkinitrd.sh /bin/
COPY kernel/kernel-source-info /etc/

COPY packages/9pudc/9pudc /sbin/
COPY packages/9pudc/etc /etc/
COPY packages/transfused/transfused /sbin/
COPY packages/transfused/etc /etc/
COPY packages/mdnstool/mdnstool /sbin/
COPY packages/mdnstool/etc /etc/
COPY packages/docker/bin/* /usr/bin/
COPY packages/docker/etc /etc/
COPY packages/docker-x/docker-x /usr/bin/
COPY packages/diagnostics/diagnostics /usr/bin/
COPY packages/diagnostics/diagnostics-server /usr/bin/
COPY packages/diagnostics/diagnostics-download /usr/bin/
COPY packages/diagnostics/etc /etc/
COPY packages/automount/etc /etc/
COPY packages/9pinit/etc /etc/
COPY packages/ntp15m/etc /etc/
COPY packages/binfmt_misc/etc /etc/
COPY packages/dnsfix/etc /etc/
COPY packages/database/etc /etc/
COPY packages/hupper/hupper /bin/
COPY packages/hupper/etc /etc/
COPY packages/hvtools/hv_fcopy_daemon /sbin/
COPY packages/hvtools/hv_kvp_daemon /sbin/
COPY packages/hvtools/hv_vss_daemon /sbin/
COPY packages/hvtools/etc /etc/
COPY packages/hvtools/hv_get_dhcp_info /sbin/
COPY packages/hvtools/hv_get_dns_info /sbin/
COPY packages/hvtools/hv_set_ifconfig /sbin/
COPY packages/userns/etc /etc/
COPY packages/userns/groupadd /usr/sbin
COPY packages/userns/useradd /usr/sbin

RUN \
  rc-update add swap boot && \
  rc-update add sysctl boot && \
  rc-update add bootmisc boot && \
  rc-update add urandom boot && \
  rc-update add hostname boot && \
  rc-update add syslog boot && \
  rc-update add networking boot && \
  rc-update add acpid default && \
  rc-update add chronyd default && \
  rc-update add cron default && \
  rc-update add savecache shutdown && \
  rc-update add killprocs shutdown && \
  rc-update add mount-ro shutdown && \
  rc-update add dmesg sysinit && \
  rc-update add devfs sysinit && \
  rc-update add mdev sysinit && \
  rc-update add hwdrivers sysinit && \
  rc-update add sysfs && \
  rc-update add fsck && \
  rc-update add root && \
  rc-update add localmount && \
  rc-update add klogd && \
  rc-update add docker default && \
  rc-update add 9pinit boot && \
  rc-update add 9pudc default && \
  rc-update add transfused default && \
  rc-update add mdnstool default && \
  rc-update add automount boot && \
  rc-update add diagnostics default && \
  rc-update add binfmt_misc sysinit && \
  rc-update add dnsfix boot && \
  rc-update add database boot && \
  rc-update add hupper default && \
  rc-update add hv_fcopy_daemon default && \
  rc-update add hv_kvp_daemon default && \
  rc-update add hv_vss_daemon default

COPY init /

CMD ["/bin/sh"]
