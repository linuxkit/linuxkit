FROM alpine:3.5

RUN \
  apk update && apk upgrade -a && \
  apk add --no-cache \
  libarchive-tools \
  && true

COPY . /

ENTRYPOINT ["/bin/sh", "-c"]
CMD ["/tartar2initrd.sh"]
