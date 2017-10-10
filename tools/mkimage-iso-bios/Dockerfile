FROM linuxkit/alpine:ad35b6ddbc70faa07e59a9d7dee7707c08122e8d

RUN \
  apk update && apk upgrade && \
  apk add --no-cache \
  libarchive-tools \
  cdrkit \
  syslinux \
  && true

COPY . .

ENTRYPOINT [ "/make-iso" ]
