FROM alpine:edge

MAINTAINER Justin Cormack <justin.cormack@unikernel.com>

COPY repositories /etc/apk

RUN apk update && apk upgrade && apk add e2fsprogs docker

COPY etc /etc/

RUN \
  setup-timezone -z UTC && \
  rc-update add networking && \
  ln -s /bin/busybox /init

CMD ["/bin/sh"]
