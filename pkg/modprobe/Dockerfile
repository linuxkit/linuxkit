# Dockerfile to build linuxkit/modprobe for linuxkit
FROM linuxkit/alpine:e9313150d2cf07ad92b0adb86799dedd9a0476e4 AS mirror

RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
RUN apk add --no-cache --initdb -p /out \
    busybox
RUN rm -rf /out/var/cache

FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=mirror /out/ /
