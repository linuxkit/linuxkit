FROM linuxkit/alpine:518c2ed0f398c5508969ac5e033607201fb419ed AS mirror

RUN apk add --no-cache go musl-dev git build-base
ENV GOPATH=/go PATH=$PATH:/go/bin 
ENV COMMIT=db7b7b0f8147f29360d69dc81af9e2877647f0de

RUN git clone https://github.com/moby/vpnkit.git /go/src/github.com/moby/vpnkit && \
    cd /go/src/github.com/moby/vpnkit && \
    git checkout $COMMIT && \
    cd go && \
    make build/vpnkit-forwarder.linux

FROM scratch
COPY --from=mirror /go/src/github.com/moby/vpnkit/go/build/vpnkit-forwarder.linux /vpnkit-forwarder
CMD ["/vpnkit-forwarder"]
