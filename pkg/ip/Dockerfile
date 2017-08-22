FROM linuxkit/alpine:a120ad6aead3fe583eaa20e9b75a05ac1b3487da AS mirror
RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
RUN apk add --no-cache --initdb -p /out \
    alpine-baselayout \
    bash \
    busybox \
    iproute2 \
    musl \
    wireguard-tools

# Remove apk residuals
RUN rm -rf /out/etc/apk /out/lib/apk /out/var/cache

FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=mirror /out/ /
LABEL org.mobyproject.config='{"capabilities": ["CAP_NET_ADMIN", "CAP_NET_RAW"]}'
