FROM linuxkit/alpine:e2391e0b164c57db9f6c4ae110ee84f766edc430 AS mirror

RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
RUN apk add --no-cache --initdb -p /out \
    --update --repository http://dl-cdn.alpinelinux.org/alpine/v3.11/main/ \
    alpine-baselayout \
    busybox \
    musl \
    avahi
RUN rm -rf /out/etc/apk /out/lib/apk /out/var/cache

FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=mirror /out/ /
COPY --from=mirror /out/var/run /var/run
COPY /avahi-daemon.conf /etc/avahi/avahi-daemon.conf
CMD ["/usr/sbin/avahi-daemon"]
