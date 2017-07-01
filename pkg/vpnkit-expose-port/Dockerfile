FROM linuxkit/alpine:9bcf61f605ef0ce36cc94d59b8eac307862de6e1 AS build

RUN apk add --no-cache go musl-dev git build-base
ENV GOPATH=/go PATH=$PATH:/go/bin 
ENV COMMIT=db7b7b0f8147f29360d69dc81af9e2877647f0de

RUN git clone https://github.com/moby/vpnkit.git /go/src/github.com/moby/vpnkit && \
    cd /go/src/github.com/moby/vpnkit && \
    git checkout $COMMIT && \
    cd go && \
    make build/vpnkit-iptables-wrapper.linux build/vpnkit-expose-port.linux

FROM scratch
COPY --from=build /go/src/github.com/moby/vpnkit/go/build/vpnkit-iptables-wrapper.linux /usr/bin/vpnkit-iptables-wrapper
COPY --from=build /go/src/github.com/moby/vpnkit/go/build/vpnkit-expose-port.linux /usr/bin/vpnkit-expose-port
