FROM linuxkit/alpine:6264e5b39af8eb1da7ffa4c05a7ccc597da01197 AS mirror

RUN apk add --no-cache go musl-dev linux-headers
ENV GOPATH=/go PATH=$PATH:/go/bin

COPY .  /go/src/kmsg/
RUN go-compile.sh /go/src/kmsg

FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=mirror /go/bin/kmsg /usr/bin/kmsg
CMD ["/usr/bin/kmsg"]
