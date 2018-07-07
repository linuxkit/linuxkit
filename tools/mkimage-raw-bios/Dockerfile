FROM linuxkit/alpine:6264e5b39af8eb1da7ffa4c05a7ccc597da01197

RUN \
  apk update && apk upgrade && \
  apk add --no-cache \
  dosfstools \
  libarchive-tools \
  sfdisk \
  syslinux \
  && true

COPY . .

ENTRYPOINT [ "/make-bios" ]
