FROM alpine:3.20
RUN apk add --no-cache zfs kmod util-linux
COPY check.sh /check.sh
ENTRYPOINT ["/bin/sh", "/check.sh"]
