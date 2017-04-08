FROM alpine:edge

RUN \
  apk update && apk upgrade && \
  apk add --no-cache \
  openssh-server \
  tini \
  util-linux \
  && true

COPY . .

RUN mkdir -p /etc/ssh /root/.ssh && chmod 0700 /root/.ssh

CMD ["/sbin/tini", "/usr/bin/ssh.sh"]
