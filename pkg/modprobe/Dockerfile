# Dockerfile to build linuxkit/modprobe for linuxkit
FROM linuxkit/alpine:8382c70bfe6b581d2c6b3032ca1081ef2ef76e1f AS mirror

RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
RUN apk add --no-cache --initdb -p /out \
    busybox
RUN rm -rf /out/var/cache

FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=mirror /out/ /
