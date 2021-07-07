FROM linuxkit/alpine:0c069d0fd7defddb6e03925fcd4915407db0c9e1 AS mirror

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
