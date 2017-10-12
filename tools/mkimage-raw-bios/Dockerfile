FROM linuxkit/alpine:87a0cd10449d72f374f950004467737dbf440630

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
