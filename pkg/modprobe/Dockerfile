# Dockerfile to build linuxkit/modprobe for linuxkit
FROM linuxkit/alpine:6090baae063eb5023c9601966e88df831f789a70 AS mirror

RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
RUN apk add --no-cache --initdb -p /out \
    busybox
RUN rm -rf /out/var/cache

FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=mirror /out/ /
