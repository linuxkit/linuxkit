FROM linuxkit/alpine:87a0cd10449d72f374f950004467737dbf440630 AS mirror

RUN apk add --no-cache go musl-dev linux-headers
ENV GOPATH=/go PATH=$PATH:/go/bin

COPY .  /go/src/metadata/
RUN go-compile.sh /go/src/metadata

RUN mkdir -p out/tmp out/var out/dev out/etc out/etc/ssl/certs

FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=mirror /go/bin/metadata /usr/bin/metadata
COPY --from=mirror /out/ /
CMD ["/usr/bin/metadata"]
LABEL org.mobyproject.config='{"binds": ["/dev:/dev", "/var:/var", "/sys:/sys", "/etc/resolv.conf:/etc/resolv.conf", "/etc/ssl/certs:/etc/ssl/certs"], "tmpfs": ["/tmp"], "readonly": true, "capabilities": ["CAP_SYS_ADMIN", "CAP_NET_ADMIN"]}'
