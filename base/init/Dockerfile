FROM alpine:3.5

RUN \
  apk update && apk upgrade -a && \
  apk add --no-cache \
  dhcpcd \
  e2fsprogs \
  e2fsprogs-extra \
  && true

COPY . ./
