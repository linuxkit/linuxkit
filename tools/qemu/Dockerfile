FROM linuxkit/alpine:77c8dfc5860012c869a19d7a2c68e701469692c8 AS mirror
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
    esac
RUN rm -rf /out/etc/apk /out/lib/apk /out/var/cache

FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=mirror /out/ /

