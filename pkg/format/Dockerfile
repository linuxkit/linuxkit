FROM linuxkit/alpine:6336329f15b4166514782eaa555cf0ffd35c519c@sha256:f6c2ce92910b1d6e4e5557850a554f4a3ae9f66c1e89ad86a24d6c6e550f165e AS mirror

RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
RUN apk add --no-cache --initdb -p /out \
    alpine-baselayout \
    busybox \
    e2fsprogs \
    e2fsprogs-extra \
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
COPY format.sh /
CMD ["/bin/sh", "/format.sh"]
