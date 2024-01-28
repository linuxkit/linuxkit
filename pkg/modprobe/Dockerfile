# Dockerfile to build linuxkit/modprobe for linuxkit
FROM linuxkit/alpine:275ef778d89133ebec5e80d82c664c3549f919d4 AS mirror

RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
RUN apk add --no-cache --initdb -p /out \
    busybox
RUN rm -rf /out/var/cache

FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=mirror /out/ /
