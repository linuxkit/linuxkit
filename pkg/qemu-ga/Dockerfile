FROM linuxkit/alpine:6584375f70524a4684f10e7155d91380420f5392 AS build
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
