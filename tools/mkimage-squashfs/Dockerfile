FROM linuxkit/alpine:f3cd219615428b2bd943411723eb28875275fae7

RUN \
  apk update && apk upgrade && \
  apk add --no-cache \
  libarchive-tools \
  squashfs-tools \
  && true

COPY . .

ENTRYPOINT [ "/make-squashfs" ]
