# Dockerfile to build linuxkit/modprobe for linuxkit
FROM linuxkit/alpine:29e84088fdd519268b05c622e4437d205acb6323 AS mirror

RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
RUN apk add --no-cache --initdb -p /out \
    busybox
RUN rm -rf /out/var/cache

FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=mirror /out/ /
