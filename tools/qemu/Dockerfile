FROM alpine:edge

COPY repositories /etc/apk/

RUN \
  apk update && apk upgrade && \
  apk add --no-cache \
  libarchive-tools \
  qemu-img \
  qemu-system-arm \
  qemu-system-x86_64 \
  ovmf@testing \
  && true

COPY . .
ENTRYPOINT ["/qemu.sh"]
