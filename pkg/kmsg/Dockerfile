FROM linuxkit/alpine:518c2ed0f398c5508969ac5e033607201fb419ed AS mirror

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
