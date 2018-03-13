FROM linuxkit/alpine:96ad1eb5ec262b4cd0eef574cdc0b225b502d9ee

RUN \
  apk update && apk upgrade && \
  apk add --no-cache \
  libarchive-tools \
  cdrkit \
  syslinux \
  && true

COPY . .

ENTRYPOINT [ "/make-iso" ]
