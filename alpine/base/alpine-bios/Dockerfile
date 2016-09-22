FROM alpine:3.4

RUN \
  apk update && apk upgrade && \
  apk add --no-cache \
  cdrkit \
  syslinux \
  && true

WORKDIR /tmp/iso

COPY isolinux.cfg ./isolinux/

COPY make-iso /usr/bin

CMD [ "/usr/bin/make-iso" ]
