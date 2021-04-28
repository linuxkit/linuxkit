FROM linuxkit/alpine:0c069d0fd7defddb6e03925fcd4915407db0c9e1 AS mirror

RUN apk add --no-cache go musl-dev git
ENV GOPATH=/go PATH=$PATH:/go/bin GO111MODULE=off

ENV VIRTSOCK_COMMIT=f1e32d3189e0dbb81c0e752a4e214617487eb41f
RUN mkdir -p $GOPATH/src/github.com/linuxkit && \
  cd $GOPATH/src/github.com/linuxkit && \
  git clone https://github.com/linuxkit/virtsock.git && \
  cd virtsock && \
  git checkout $VIRTSOCK_COMMIT

COPY .  /go/src/host-timesync-daemon
RUN go-compile.sh /go/src/host-timesync-daemon

FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=mirror /go/bin/host-timesync-daemon /usr/bin/host-timesync-daemon
CMD ["/usr/bin/host-timesync-daemon", "-port", "0xf3a4"]
