FROM alpine:3.5

RUN \
  apk update && apk upgrade && \
  apk add --no-cache \
  qemu-img \
  qemu-system-arm \
  qemu-system-x86_64 \
  && true
