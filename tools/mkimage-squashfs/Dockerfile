FROM linuxkit/alpine:5ac048802293564a50414394e1b9896eac528e7d AS mirror
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
