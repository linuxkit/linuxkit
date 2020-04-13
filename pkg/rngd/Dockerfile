FROM linuxkit/alpine:bc528cf9d4065d2e09aa44ff76909b94cfe8d867 AS mirror

RUN apk add --no-cache go gcc musl-dev linux-headers
ENV GOPATH=/go PATH=$PATH:/go/bin

# see https://github.com/golang/go/issues/23672
ENV CGO_CFLAGS_ALLOW=(-mrdrnd|-mrdseed)

COPY cmd/rngd/ /go/src/rngd/
RUN REQUIRE_CGO=1 go-compile.sh /go/src/rngd

FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=mirror /go/bin/rngd /sbin/rngd
CMD ["/sbin/rngd"]
