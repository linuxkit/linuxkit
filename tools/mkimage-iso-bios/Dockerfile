FROM --platform=linux/amd64 linuxkit/alpine:e9313150d2cf07ad92b0adb86799dedd9a0476e4 as amd64-builder
RUN apk add --no-cache syslinux
RUN mkdir /amd64-out && \
    cp /usr/share/syslinux/isolinux.bin /amd64-out/ && \
    cp /usr/share/syslinux/ldlinux.c32 /amd64-out/ && \
    cp /usr/bin/isohybrid /amd64-out/

FROM linuxkit/alpine:e9313150d2cf07ad92b0adb86799dedd9a0476e4 as mirror
RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
RUN apk add --no-cache --initdb -p /out \
    alpine-baselayout \
    busybox \
    cdrkit \
    libarchive-tools \
    && true
RUN mv /out/etc/apk/repositories.upstream /out/etc/apk/repositories

FROM scratch
WORKDIR /
COPY --from=mirror /out/ /
RUN mkdir -p /isolinux
COPY --from=amd64-builder /amd64-out/isolinux.bin /syslinux/
COPY --from=amd64-builder /amd64-out/ldlinux.c32 /syslinux/
COPY --from=amd64-builder /amd64-out/isohybrid /syslinux/
COPY . .
ENTRYPOINT [ "/make-iso" ]

