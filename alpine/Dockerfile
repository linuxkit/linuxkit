FROM alpine:edge

MAINTAINER Justin Cormack <justin.cormack@unikernel.com>

COPY repositories /etc/apk

RUN apk update && apk upgrade && apk add e2fsprogs docker chrony

COPY etc /etc/

COPY mkinitrd.sh /bin/

RUN \
  setup-timezone -z UTC && \
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
  ln -s /bin/busybox /init

CMD ["/bin/sh"]
