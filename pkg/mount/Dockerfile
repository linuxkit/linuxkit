FROM linuxkit/alpine:488aa6f5dd2d8121a3c5c5c7a1ecf97c424b96ac AS mirror

RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
RUN apk add --no-cache --initdb -p /out \
    alpine-baselayout \
    busybox \
    musl \
    sfdisk \
    && true
RUN rm -rf /out/etc/apk /out/lib/apk /out/var/cache

FROM linuxkit/alpine:488aa6f5dd2d8121a3c5c5c7a1ecf97c424b96ac AS build

RUN apk add --no-cache go musl-dev
ENV GOPATH=/go PATH=$PATH:/go/bin

COPY *.go /go/src/mountie/
RUN go-compile.sh /go/src/mountie

FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=mirror /out/ /
COPY --from=build /go/bin/mountie usr/bin/mountie
CMD ["/usr/bin/mountie"]
LABEL org.mobyproject.config='{"binds": ["/dev:/dev", "/var:/var:rshared,rbind", "/:/hostroot"], "capabilities": ["CAP_SYS_ADMIN"], "rootfsPropagation": "shared", "net": "new", "ipc": "new"}'
