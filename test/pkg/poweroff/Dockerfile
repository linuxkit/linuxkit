FROM linuxkit/alpine:3683c9a66cd4da40bd7d6c7da599b2dcd738b559 AS mirror
RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
RUN apk add --no-cache --initdb -p /out alpine-baselayout busybox musl
RUN rm -rf /out/etc/apk /out/lib/apk /out/var/cache

FROM scratch
CMD []
WORKDIR /
COPY --from=mirror /out/ /
COPY /poweroff.sh /poweroff.sh
ENTRYPOINT ["/bin/sh", "/poweroff.sh"]
LABEL org.mobyproject.config='{"pid": "host", "readonly": true, "capabilities": ["CAP_SYS_BOOT"]}'
