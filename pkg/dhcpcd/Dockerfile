FROM alpine:3.5

RUN \
  apk update && apk upgrade -a && \
  apk add --no-cache \
  dhcpcd \
  openrc \
  && true

ADD dhcpcd.conf /dhcpcd.conf
ADD start_dhcpcd.sh /usr/local/bin/start_dhcpcd.sh
ADD 10-mtu /usr/lib/dhcpcd/dhcpcd-hooks/10-mtu

CMD ["/usr/local/bin/start_dhcpcd.sh"]