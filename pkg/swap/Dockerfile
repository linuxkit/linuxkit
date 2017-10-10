FROM linuxkit/alpine:ad35b6ddbc70faa07e59a9d7dee7707c08122e8d AS mirror

RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
RUN apk add --no-cache --initdb -p /out \
    alpine-baselayout \
    busybox \
    cryptsetup \
    musl \
    util-linux
RUN rm -rf /out/etc/apk /out/lib/apk /out/var/cache

FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=mirror /out/ /
COPY /swap.sh .
ENTRYPOINT ["swap.sh"]
LABEL org.mobyproject.config='{"binds": ["/dev:/dev", "/var:/var"], "capabilities": ["CAP_SYS_ADMIN", "CAP_MKNOD"], "net": "new", "ipc": "new"}'
