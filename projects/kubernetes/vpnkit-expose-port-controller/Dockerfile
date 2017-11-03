FROM linuxkit/alpine:3fb44354a34b05134fbf585a00217cd2f8c8f0bf AS build
RUN apk add --no-cache go musl-dev git build-base
ENV GOPATH=/go PATH=$PATH:/go/bin
COPY . /go/src/vpnkit-expose-port-controller
RUN go-compile.sh /go/src/vpnkit-expose-port-controller

FROM linuxkit/vpnkit-expose-port:fa4ab4ac78b83fe392e39b861b4114c3bb02d170 AS vpnkit

FROM scratch
WORKDIR /
ENTRYPOINT ["/usr/bin/vpnkit-expose-port-controller"]
COPY --from=build /go/bin/vpnkit-expose-port-controller /usr/bin/vpnkit-expose-port-controller
COPY --from=vpnkit /usr/bin/vpnkit-expose-port /usr/bin/vpnkit-expose-port
