FROM linuxkit/alpine:ad35b6ddbc70faa07e59a9d7dee7707c08122e8d AS mirror

RUN apk add --no-cache go gcc musl-dev linux-headers
ENV GOPATH=/go PATH=$PATH:/go/bin

COPY cmd/rngd/ /go/src/rngd/
RUN REQUIRE_CGO=1 go-compile.sh /go/src/rngd

FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=mirror /go/bin/rngd /sbin/rngd
CMD ["/sbin/rngd"]
LABEL org.mobyproject.config='{"capabilities": ["CAP_SYS_ADMIN"], "oomScoreAdj": -800, "readonly": true, "net": "new", "ipc": "new"}'
