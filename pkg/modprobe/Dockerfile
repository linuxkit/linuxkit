# Dockerfile to build linuxkit/modprobe for linuxkit
FROM linuxkit/alpine:a720d2b1b83556653abd52d61faa0fe020c2b161 AS mirror

RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
RUN apk add --no-cache --initdb -p /out \
    busybox
RUN rm -rf /out/var/cache

FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=mirror /out/ /
