FROM alpine:3.4

RUN \
  apk update && apk upgrade && \
  apk add \
  alpine-conf \
  bind-tools \
  busybox-initscripts \
  chrony \
  cifs-utils \
  dhcpcd \
  e2fsprogs \
  e2fsprogs-extra \
  fuse \
  git \
  iptables \
  jq \
  lvm2 \
  make \
  openrc \
  openssh-client \
  openssl \
  sfdisk \
  strace \
  sysklogd \
  syslinux \
  tar \
  xz \
  && rm -rf /var/cache/apk/*
