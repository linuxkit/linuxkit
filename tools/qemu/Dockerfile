FROM linuxkit/alpine:5240cbd9cf371c8211c8f1968e57c51a32098c8f AS mirror
RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
RUN apk add --no-cache --initdb -p /out \
    alpine-baselayout \
    busybox \
    libarchive-tools \
    qemu-img \
    qemu-system-x86_64

RUN mkdir -p /out/usr/share/ovmf \
    && cp /usr/share/ovmf/bios.bin /out/usr/share/ovmf/bios.bin
RUN rm -rf /out/etc/apk /out/lib/apk /out/var/cache

FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=mirror /out/ /

