FROM linuxkit/alpine:146f540f25cd92ec8ff0c5b0c98342a9a95e479e AS build
RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
RUN mkdir -p /out/var/run
RUN apk add --no-cache --initdb -p /out \
    busybox \
    qemu-guest-agent \
    musl
FROM scratch
WORKDIR /
ENTRYPOINT []
COPY --from=build /out /
CMD ["/usr/bin/qemu-ga", "-p", "/dev/vport0p1"]
