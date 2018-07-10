FROM linuxkit/alpine:daed76b8f1d28cdeeee215a95b9671c682a405dc AS mirror
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
