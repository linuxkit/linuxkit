FROM linuxkit/alpine:87a0cd10449d72f374f950004467737dbf440630 AS build
RUN apk add --no-cache --initdb alpine-baselayout make gcc musl-dev

ADD usermode-helper.c .
RUN make usermode-helper

RUN apk add --no-cache go musl-dev
ENV GOPATH=/go PATH=$PATH:/go/bin

COPY init.go /go/src/init/
COPY vendor /go/src/init/vendor/
RUN go-compile.sh /go/src/init/

FROM linuxkit/alpine:87a0cd10449d72f374f950004467737dbf440630 AS mirror
RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
RUN apk add --no-cache --initdb -p /out alpine-baselayout busybox musl

# Remove apk residuals. We have a read-only rootfs, so apk is of no use.
RUN rm -rf /out/etc/apk /out/lib/apk /out/var/cache

FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=build /go/bin/init /
COPY --from=build usermode-helper /sbin/
COPY --from=mirror /out/ /
COPY etc etc/
COPY bin bin/
