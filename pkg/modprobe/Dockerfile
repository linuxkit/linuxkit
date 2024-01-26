# Dockerfile to build linuxkit/modprobe for linuxkit
FROM linuxkit/alpine:6584375f70524a4684f10e7155d91380420f5392 AS mirror

RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
RUN apk add --no-cache --initdb -p /out \
    busybox
RUN rm -rf /out/var/cache

FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=mirror /out/ /
