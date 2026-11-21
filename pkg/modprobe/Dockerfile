# Dockerfile to build linuxkit/modprobe for linuxkit
FROM linuxkit/alpine:d497d5be5b78bfd78268f3466cc4080aee5d7389 AS mirror

RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
RUN apk add --no-cache --initdb -p /out \
    busybox
RUN rm -rf /out/var/cache

FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=mirror /out/ /
