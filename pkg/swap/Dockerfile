FROM linuxkit/alpine:316c3f9d85c21fdd8bc7479e81d290f85bf60eb0 AS mirror

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
