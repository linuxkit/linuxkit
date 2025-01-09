# Dockerfile to build linuxkit/modprobe for linuxkit
FROM linuxkit/alpine:5ac048802293564a50414394e1b9896eac528e7d AS mirror

RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
RUN apk add --no-cache --initdb -p /out \
    busybox
RUN rm -rf /out/var/cache

FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=mirror /out/ /
