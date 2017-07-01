FROM linuxkit/alpine:9bcf61f605ef0ce36cc94d59b8eac307862de6e1 AS mirror

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
