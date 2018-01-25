FROM linuxkit/alpine:cba395fbc278daee841106801aba1e1bd7e0f2f7 AS mirror

RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
RUN apk add --no-cache --initdb -p /out \
    alpine-baselayout \
    busybox \
    musl \
    sfdisk \
    && true
RUN rm -rf /out/etc/apk /out/lib/apk /out/var/cache

FROM linuxkit/alpine:cba395fbc278daee841106801aba1e1bd7e0f2f7 AS build

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
