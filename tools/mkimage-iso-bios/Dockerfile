FROM linuxkit/alpine:d307c8a386fa3f32cddda9409b9687e191cdd6f1

RUN \
  apk update && apk upgrade && \
  apk add --no-cache \
  libarchive-tools \
  cdrkit \
  syslinux \
  && true

COPY . .

ENTRYPOINT [ "/make-iso" ]
