FROM alpine:edge

MAINTAINER Justin Cormack <justin.cormack@unikernel.com>

COPY repositories /etc/apk/repositories
COPY motd hostname /etc/

RUN apk update && apk upgrade && apk add e2fsprogs docker

CMD ["/bin/sh"]
