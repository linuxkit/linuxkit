FROM alpine:3.5

COPY repositories /etc/apk/

RUN \
  apk update && apk upgrade -a && \
  apk add --no-cache \
  dhcpcd \
  e2fsprogs \
  e2fsprogs-extra \
  policycoreutils \
  libselinux-utils \
  && true

COPY . ./
