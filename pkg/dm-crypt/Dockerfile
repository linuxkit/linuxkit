FROM linuxkit/alpine:bc528cf9d4065d2e09aa44ff76909b94cfe8d867 AS mirror
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
