FROM linuxkit/alpine:146f540f25cd92ec8ff0c5b0c98342a9a95e479e AS mirror
RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
RUN apk add curl
RUN apk add --no-cache --initdb -p /out \
    alpine-baselayout \
    bash \
    busybox \
    iproute2 \
    iptables \
    ebtables \
    ipvsadm \
    bridge-utils \
    musl \
    wireguard-tools

# We grab our version of wg-quick, called lk-wg-config.sh
RUN curl -sSL -o /out/usr/bin/lk-wg-config https://gist.githubusercontent.com/zx2c4/3624de869ab7eaef3de5ea8f2b867be9/raw/de72b018f4f4548858ce6aae2898b34db0474221/lk-wg-config.sh && chmod 755 /out/usr/bin/lk-wg-config

# Remove apk residuals
RUN rm -rf /out/etc/apk /out/lib/apk /out/var/cache

FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=mirror /out/ /
