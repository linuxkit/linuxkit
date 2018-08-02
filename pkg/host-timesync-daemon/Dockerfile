FROM linuxkit/alpine:3683c9a66cd4da40bd7d6c7da599b2dcd738b559 AS mirror

RUN apk add --no-cache go musl-dev git
ENV GOPATH=/go PATH=$PATH:/go/bin

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
