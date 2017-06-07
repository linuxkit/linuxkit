FROM linuxkit/alpine:630ee558e4869672fae230c78364e367b8ea67a9 AS mirror

RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
RUN apk add --no-cache --initdb -p /out \
    alpine-baselayout \
    busybox \
    musl \
    tini \
    util-linux \
    && true
RUN rm -rf /out/etc/apk /out/lib/apk /out/var/cache

FROM scratch
ENTRYPOINT ["/sbin/tini","-s","-v","--"]
WORKDIR /
COPY --from=mirror /out/ /
COPY usr/ /usr/
COPY etc/ /etc/
CMD ["/usr/bin/rungetty.sh"]
LABEL org.mobyproject.config='{"pid": "host", "net":"host", "binds": ["/run:/run", "/etc:/hostroot/etc","/tmp/ctr:/tmp/ctr", "/usr/bin/ctr:/usr/bin/ctr", "/usr/bin/runc:/usr/bin/runc", "/usr/bin/dist:/usr/bin/dist", "/var:/var","/containers:/containers","/dev:/dev"], "capabilities": ["all"]}'
