FROM alpine:edge

RUN \
  apk update && apk upgrade && \
  apk add --no-cache \
  dosfstools \
  libarchive-tools \
  binutils \
  gummiboot \
  mtools \
  xorriso \
  && true

COPY . .

ENTRYPOINT [ "/make-efi" ]
