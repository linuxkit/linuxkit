FROM linuxkit/alpine:c608d404c1cb36cce0c7d9303e30b52c9d81ccf0 AS mirror

RUN apk add --no-cache go musl-dev
ENV GOPATH=/go PATH=$PATH:/go/bin

COPY .  /go/src/metadata/
RUN go-compile.sh /go/src/metadata

FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=mirror /go/bin/metadata /usr/bin/metadata
CMD ["/usr/bin/metadata"]
LABEL org.mobyproject.config='{"binds": ["/dev:/dev", "/var:/var", "/tmp/etc/resolv.conf:/etc/resolv.conf"], "capabilities": ["CAP_SYS_ADMIN"]}'
