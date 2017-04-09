FROM alpine:3.5

RUN \
  apk update && apk upgrade -a && \
  apk add --no-cache \
  jq \
  sfdisk \
  && true

COPY . ./
CMD ["/bin/sh", "/mount.sh"]
