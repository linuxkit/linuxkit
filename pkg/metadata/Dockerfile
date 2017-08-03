FROM linuxkit/alpine:2e362f4459ba4491655061cccdd2fcc7a2de5eb3 AS mirror

RUN apk add --no-cache go musl-dev
ENV GOPATH=/go PATH=$PATH:/go/bin

COPY .  /go/src/metadata/
RUN go-compile.sh /go/src/metadata

RUN mkdir -p out/tmp out/var out/dev out/etc

FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=mirror /go/bin/metadata /usr/bin/metadata
COPY --from=mirror /out/ /
CMD ["/usr/bin/metadata"]
LABEL org.mobyproject.config='{"binds": ["/dev:/dev", "/var:/var", "/etc/resolv.conf:/etc/resolv.conf"], "tmpfs": ["/tmp"], "readonly": true, "capabilities": ["CAP_SYS_ADMIN"]}'
