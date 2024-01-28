FROM linuxkit/alpine:7aea0bb7faf70f35c6b7d2d244c91da1d4b3ea93 AS mirror
RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
RUN apk add --no-cache --initdb -p /out \
  alpine-baselayout \
  busybox \
  libarchive-tools \
  squashfs-tools \
  && true
RUN mv /out/etc/apk/repositories.upstream /out/etc/apk/repositories

FROM scratch
WORKDIR /
COPY --from=mirror /out/ /
COPY . .
ENTRYPOINT [ "/make-squashfs" ]
