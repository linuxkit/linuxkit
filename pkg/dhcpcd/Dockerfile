FROM alpine:3.5

RUN \
  apk update && apk upgrade -a && \
  apk add --no-cache \
  dhcpcd \
  openrc \
  && true

COPY . .

CMD ["/sbin/dhcpcd", "--nobackground", "-f", "/dhcpcd.conf"]
