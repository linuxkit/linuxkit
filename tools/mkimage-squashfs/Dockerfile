FROM linuxkit/alpine:33063834cf72d563cd8703467836aaa2f2b5a300 AS mirror
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
