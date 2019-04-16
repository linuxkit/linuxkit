FROM linuxkit/alpine:86cd4f51b49fb9a078b50201d892a3c7973d48ec AS mirror
RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
RUN apk add --no-cache --initdb -p /out \
    alpine-baselayout \
    busybox \
    libarchive-tools \
    qemu-img && \
    case $(uname -m) in \
    x86_64) \
        apk add --no-cache --initdb -p /out qemu-system-x86_64 ovmf; \
        ;; \
    aarch64) \
        apk add --no-cache --initdb -p /out qemu-system-aarch64; \
        ;; \
    s390x) \
        apk add --no-cache --initdb -p /out qemu-system-s390x; \
        ;; \
    esac
RUN rm -rf /out/etc/apk /out/lib/apk /out/var/cache

FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=mirror /out/ /

