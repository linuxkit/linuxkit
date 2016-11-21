FROM alpine:3.4

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
  hvtools@nextmain \
  iptables \
  jq \
  openrc \
  openssh \
  openssh-client \
  rng-tools@edgecommunity \
  sfdisk \
  sysklogd \
  xz \
  && true
