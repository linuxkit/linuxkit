FROM alpine:edge

RUN \
  apk update && apk upgrade && \
  apk add \
  qemu-img \
  qemu-system-arm \
  qemu-system-x86_64 \
  && rm -rf /var/cache/apk/*
