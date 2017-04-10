FROM alpine:3.5

RUN \
  apk update && apk upgrade -a && \
  apk add --no-cache \
  openntpd \
  && true

COPY . .

CMD ["/usr/sbin/ntpd", "-d", "-s"]
