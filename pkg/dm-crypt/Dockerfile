FROM linuxkit/alpine:518c2ed0f398c5508969ac5e033607201fb419ed AS mirror
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
