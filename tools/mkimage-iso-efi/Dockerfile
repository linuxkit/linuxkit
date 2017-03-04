FROM alpine:3.5

RUN \
  apk update && apk upgrade && \
  apk add --no-cache \
  libarchive-tools \
  binutils \
  gummiboot \
  mtools \
  xorriso \
  && true

COPY . .

CMD [ "/make-efi" ]
