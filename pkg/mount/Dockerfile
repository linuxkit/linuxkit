FROM linuxkit/containerd:5749f2e9e65395cc6635229e8da0e0d484320ddf AS mirror

RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
RUN apk add --no-cache --initdb -p /out \
    alpine-baselayout \
    busybox \
    jq \
    musl \
    sfdisk \
    && true
RUN rm -rf /out/etc/apk /out/lib/apk /out/var/cache

FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=mirror /out/ /
COPY mount.sh /
CMD ["/bin/sh", "/mount.sh"]
LABEL org.mobyproject.config='{"binds": ["/dev:/dev", "/var:/var:rshared,rbind"], "capabilities": ["CAP_SYS_ADMIN"], "rootfsPropagation": "shared", "net": "new", "ipc": "new"}'
