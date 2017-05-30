FROM linuxkit/alpine:630ee558e4869672fae230c78364e367b8ea67a9 AS mirror
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
LABEL org.mobyproject.config='{"binds": ["/var:/var", "/tmp/etc:/etc"], "capabilities": ["CAP_NET_ADMIN", "CAP_NET_BIND_SERVICE", "CAP_NET_RAW"]}'
