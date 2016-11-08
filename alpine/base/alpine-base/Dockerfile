FROM alpine:3.4

COPY repositories /etc/apk/

RUN \
  apk update && apk upgrade && \
  apk add --no-cache \
  alpine-conf \
  busybox-initscripts \
  chrony \
  cifs-utils \
  curl \
  dhcpcd \
  e2fsprogs \
  e2fsprogs-extra \
  fuse \
  git \
  hvtools@nextmain \
  iptables \
  ipvsadm@nextmain \
  jq \
  make \
  openrc \
  openssh \
  openssh-client \
  openssl \
  rng-tools@edgecommunity \
  sfdisk \
  strace \
  sysklogd \
  tar \
  xz \
  && true
