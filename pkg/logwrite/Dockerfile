FROM linuxkit/alpine:3683c9a66cd4da40bd7d6c7da599b2dcd738b559 AS build

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
