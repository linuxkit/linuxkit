FROM alpine:3.5

RUN \
  apk update && apk upgrade -a && \
  apk add --no-cache \
  e2fsprogs \
  e2fsprogs-extra \
  jq \
  sfdisk \
  && true

COPY . ./
CMD ["/bin/sh", "/format.sh"]
