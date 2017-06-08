FROM linuxkit/alpine:630ee558e4869672fae230c78364e367b8ea67a9 AS mirror

RUN apk add --no-cache go musl-dev git build-base
ENV GOPATH=/go PATH=$PATH:/go/bin 
ENV COMMIT=2d6d82167cf81c665c05d1425a79adfbc1a71177

RUN git clone https://github.com/moby/vpnkit.git /go/src/github.com/moby/vpnkit && \
    cd /go/src/github.com/moby/vpnkit && \
    git checkout $COMMIT && \
    cd go && \
    make all

FROM scratch
COPY --from=mirror /go/src/github.com/moby/vpnkit/go/build/vpnkit-forwarder.linux /vpnkit-forwarder
COPY --from=mirror /go/src/github.com/moby/vpnkit/go/build/vpnkit-expose-port.linux /vpnkit-expose-port
CMD ["/vpnkit-forwarder"]
