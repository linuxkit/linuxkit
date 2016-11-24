FROM alpine:3.4

COPY repositories /etc/apk/

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
  openrc \
  openssh-client \
  rng-tools@community \
  sfdisk \
  sysklogd \
  xz \
  && true
