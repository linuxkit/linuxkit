FROM linuxkit/alpine:87a0cd10449d72f374f950004467737dbf440630 AS mirror
RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
RUN apk add --no-cache --initdb -p /out \
    alpine-baselayout \
    busybox \
    dhcpcd \
    musl

# Remove apk residuals
RUN rm -rf /out/etc/apk /out/lib/apk /out/var/cache

FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=mirror /out/ /
COPY /dhcpcd.conf /usr/ /
CMD ["/sbin/dhcpcd", "--nobackground", "-f", "/dhcpcd.conf"]
LABEL org.mobyproject.config='{"binds": ["/run/resolvconf:/etc"], "capabilities": ["CAP_NET_ADMIN", "CAP_NET_BIND_SERVICE", "CAP_NET_RAW"]}'
