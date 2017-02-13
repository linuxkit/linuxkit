FROM alpine:3.5

COPY repositories /etc/apk/
COPY selinux-config /etc/selinux/config

RUN \
  apk update && apk upgrade -a && \
  apk add --no-cache \
  busybox-initscripts \
  chrony \
  cifs-utils \
  curl \
  dhcpcd \
  e2fsprogs \
  e2fsprogs-extra \
  fuse \
  git \
  hvtools \
  iptables \
  jq \
  libselinux \
  libsepol \
  libselinux-utils \
  openrc \
  openssh-client \
  sfdisk \
  strace \
  sysklogd \
  xz \
  && true
