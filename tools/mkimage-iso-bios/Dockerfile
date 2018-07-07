FROM linuxkit/alpine:6264e5b39af8eb1da7ffa4c05a7ccc597da01197

RUN \
  apk update && apk upgrade && \
  apk add --no-cache \
  libarchive-tools \
  cdrkit \
  syslinux \
  && true

COPY . .

ENTRYPOINT [ "/make-iso" ]
