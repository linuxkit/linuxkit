FROM linuxkit/alpine:0c069d0fd7defddb6e03925fcd4915407db0c9e1 AS mirror

RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
RUN apk add --no-cache --initdb -p /out \
    alpine-baselayout \
    busybox \
    e2fsprogs \
    e2fsprogs-extra \
    musl \
    sfdisk \
    syslinux \
    xfsprogs \
    && true
RUN rm -rf /out/etc/apk /out/lib/apk /out/var/cache

FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=mirror /out/ /
COPY mkimage.sh /usr/bin/
CMD ["mkimage.sh"]
