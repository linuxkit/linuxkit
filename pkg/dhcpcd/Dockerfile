FROM linuxkit/alpine:5f6db26ab7bf6a9c452a612e236cc7495408132b@sha256:d009afc85d0b005daf51c8f3026aa552ab997dc47cab43915e9dc761accae086 AS mirror
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
LABEL org.mobyproject.config='{"binds": ["/var:/var", "/tmp/etc:/etc"], "net": "host", "capabilities": ["CAP_NET_ADMIN", "CAP_NET_BIND_SERVICE", "CAP_NET_RAW"]}'
