FROM linuxkit/alpine:07f7d136e427dc68154cd5edbb2b9576f9ac5213

RUN \
  apk update && apk upgrade && \
  apk add --no-cache \
  libarchive-tools \
  cdrkit \
  syslinux \
  && true

COPY . .

ENTRYPOINT [ "/make-iso" ]
