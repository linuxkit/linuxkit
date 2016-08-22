FROM alpine:3.4

RUN \
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
  cifs-utils \
  e2fsprogs-extra \
  openssl \
  jq \
  util-linux \
  && rm -rf /var/cache/apk/*
