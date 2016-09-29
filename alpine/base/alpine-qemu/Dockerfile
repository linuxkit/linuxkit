FROM alpine:3.4

COPY repositories /etc/apk/

RUN \
  apk update && apk upgrade && \
  apk add --no-cache \
  qemu-img@edge \
  qemu-system-arm@edge \
  qemu-system-x86_64@edge \
  && true
