FROM linuxkit/alpine:8382c70bfe6b581d2c6b3032ca1081ef2ef76e1f AS mirror

RUN apk add --no-cache go musl-dev git
ENV GOPATH=/go PATH=$PATH:/go/bin
ARG GOPKGVERSION
ENV ldflags="-X main.Version=$GOPKGVERSION"

COPY . /go/src/host-timesync-daemon
RUN go-compile.sh /go/src/host-timesync-daemon

FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=mirror /go/bin/host-timesync-daemon /usr/bin/host-timesync-daemon
CMD ["/usr/bin/host-timesync-daemon", "-port", "0xf3a4"]
