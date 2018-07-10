FROM linuxkit/alpine:daed76b8f1d28cdeeee215a95b9671c682a405dc

RUN \
  apk update && apk upgrade && \
  apk add --no-cache \
  libarchive-tools \
  squashfs-tools \
  && true

COPY . .

ENTRYPOINT [ "/make-squashfs" ]
