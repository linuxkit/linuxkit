FROM linuxkit/alpine:bc528cf9d4065d2e09aa44ff76909b94cfe8d867 AS build

RUN apk add --no-cache go musl-dev
ENV GOPATH=/go PATH=$PATH:/go/bin

COPY logwrite.go /go/src/logwrite/
RUN go-compile.sh /go/src/logwrite

FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=build /go/bin/logwrite usr/bin/logwrite
CMD ["/usr/bin/logwrite"]
