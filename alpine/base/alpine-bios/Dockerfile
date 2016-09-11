FROM alpine:3.4

RUN \
  apk update && apk upgrade && \
  apk add \
  cdrkit \
  syslinux \
  && rm -rf /var/cache/apk/*
