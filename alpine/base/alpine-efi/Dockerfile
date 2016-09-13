FROM alpine:3.4

RUN \
  apk update && apk upgrade && \
  apk add \
  binutils \
  mtools \
  xorriso \
  gummiboot \
  && rm -rf /var/cache/apk/*
