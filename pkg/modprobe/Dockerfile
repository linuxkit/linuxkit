# Dockerfile to build linuxkit/modprobe for linuxkit
FROM linuxkit/alpine:3fb2ec0c94ed30171bb04100f568bba2ebb648a9 AS mirror

RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
RUN apk add --no-cache --initdb -p /out \
    busybox
RUN rm -rf /out/var/cache

FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=mirror /out/ /
