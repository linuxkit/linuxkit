FROM linuxkit/alpine:86cd4f51b49fb9a078b50201d892a3c7973d48ec as mirror
RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
RUN apk add --no-cache --initdb -p /out \
    alpine-baselayout \
    busybox \
    cdrkit \
    libarchive-tools \
    syslinux \
    && true
RUN mv /out/etc/apk/repositories.upstream /out/etc/apk/repositories

FROM scratch
WORKDIR /
COPY --from=mirror /out/ /
COPY . .
ENTRYPOINT [ "/make-iso" ]
