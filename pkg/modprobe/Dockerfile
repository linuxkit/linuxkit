# Dockerfile to build linuxkit/modprobe for linuxkit
FROM linuxkit/alpine:7f3944798557de5518a56e3437d7ed982701f224 AS mirror

RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
RUN apk add --no-cache --initdb -p /out \
    busybox
RUN rm -rf /out/var/cache

FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=mirror /out/ /
