FROM linuxkit/alpine:9bcf61f605ef0ce36cc94d59b8eac307862de6e1 AS mirror
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

