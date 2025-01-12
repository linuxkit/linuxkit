# Dockerfile to build linuxkit/modprobe for linuxkit
FROM linuxkit/alpine:8c02670fd2e5c28d03fd39db0bddc1379b3af6b2 AS mirror

RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
RUN apk add --no-cache --initdb -p /out \
    busybox
RUN rm -rf /out/var/cache

FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=mirror /out/ /
