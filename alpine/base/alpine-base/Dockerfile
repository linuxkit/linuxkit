FROM alpine:edge

COPY repositories /etc/apk/

RUN \
  apk update && apk upgrade && \
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
  openrc \
  openssh \
  openssh-client \
  rng-tools@community \
  sfdisk \
  sysklogd \
  xz \
  && true
