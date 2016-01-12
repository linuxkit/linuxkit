FROM alpine:3.3

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
  strace fuse

COPY etc /etc/

ADD kernel/aufs-utils.tar /
COPY mkinitrd.sh /bin/

COPY packages/9pudc/9pudc /sbin/
COPY packages/9pudc/etc /etc/
COPY packages/9pudfuse/9pudfuse /sbin/
COPY packages/9pudfuse/etc /etc/
COPY packages/mdnstool/mdnstool /sbin/
COPY packages/mdnstool/etc /etc/
COPY packages/docker/docker /usr/bin/
COPY packages/docker/etc /etc/
COPY packages/docker-x/docker-x /usr/bin/
COPY packages/diagnostics/diagnostics /usr/bin/
COPY packages/diagnostics/etc /etc/
COPY packages/automount/etc /etc/
COPY packages/9pinit/etc /etc/
COPY packages/ntp15m/etc /etc/
COPY packages/binfmt_misc/etc /etc/

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
  rc-update add 9pinit default && \
  rc-update add 9pudc default && \
  rc-update add 9pudfuse default && \
  rc-update add mdnstool default && \
  rc-update add automount boot && \
  rc-update add diagnostics default && \
  rc-update add binfmt_misc sysinit && \
  ln -s /bin/busybox /init

CMD ["/bin/sh"]
