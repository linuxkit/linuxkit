FROM alpine:edge

RUN \
  apk update && apk upgrade && \
  apk add \
  binutils \
  mtools \
  xorriso \
  gummiboot \
  && rm -rf /var/cache/apk/*
