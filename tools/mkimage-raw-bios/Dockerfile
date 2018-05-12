FROM linuxkit/alpine:1b05307ae8152e3d38f79e297b0632697a30c65c

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
