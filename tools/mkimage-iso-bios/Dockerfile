FROM linuxkit/alpine:cba395fbc278daee841106801aba1e1bd7e0f2f7

RUN \
  apk update && apk upgrade && \
  apk add --no-cache \
  libarchive-tools \
  cdrkit \
  syslinux \
  && true

COPY . .

ENTRYPOINT [ "/make-iso" ]
