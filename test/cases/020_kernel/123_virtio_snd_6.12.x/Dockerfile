FROM alpine:3.22
RUN apk add --no-cache kmod
COPY check.sh /check.sh
ENTRYPOINT ["/bin/sh", "/check.sh"]
