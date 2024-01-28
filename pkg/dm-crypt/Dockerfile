FROM linuxkit/alpine:275ef778d89133ebec5e80d82c664c3549f919d4 AS mirror
RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
RUN apk add --no-cache --initdb -p /out  \
    alpine-baselayout \
    cryptsetup \
    e2fsprogs

# Remove apk residuals
RUN rm -rf /out/etc/apk /out/lib/apk /out/var/cache

FROM scratch
ENTRYPOINT []
WORKDIR /
COPY --from=mirror /out/ /

COPY crypto.sh /usr/bin/crypto
RUN chmod +x /usr/bin/crypto

CMD ["/usr/bin/crypto"]
