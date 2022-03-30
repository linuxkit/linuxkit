FROM linuxkit/alpine:33063834cf72d563cd8703467836aaa2f2b5a300 AS mirror

RUN apk add --no-cache go musl-dev git
ENV GOPATH=/go PATH=$PATH:/go/bin

COPY . /go/src/host-timesync-daemon
RUN go-compile.sh /go/src/host-timesync-daemon

FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=mirror /go/bin/host-timesync-daemon /usr/bin/host-timesync-daemon
CMD ["/usr/bin/host-timesync-daemon", "-port", "0xf3a4"]
