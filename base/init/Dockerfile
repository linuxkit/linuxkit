FROM alpine:3.5

RUN echo http://dl-cdn.alpinelinux.org/alpine/edge/testing >> /etc/apk/repositories
RUN \
  apk --no-cache update && \
  apk --no-cache upgrade -a && \
  apk --no-cache add \
  dhcpcd \
  e2fsprogs \
  e2fsprogs-extra \
  policycoreutils \
  && rm -rf /var/cache/apk/*

COPY . ./
