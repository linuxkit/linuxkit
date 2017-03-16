FROM alpine:3.5

RUN \
  apk --no-cache update && \
  apk --no-cache upgrade -a && \
  apk --no-cache add \
  dhcpcd \
  e2fsprogs \
  e2fsprogs-extra \
  && rm -rf /var/cache/apk/*

COPY . ./
