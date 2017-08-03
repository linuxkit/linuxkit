FROM linuxkit/alpine:87a0cd10449d72f374f950004467737dbf440630 AS mirror

RUN apk add --no-cache go musl-dev
ENV GOPATH=/go PATH=$PATH:/go/bin

COPY main.go /go/src/sysctl/
RUN go-compile.sh /go/src/sysctl

FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=mirror /go/bin/sysctl /usr/bin/sysctl
COPY etc/ /etc/
CMD ["/usr/bin/sysctl"]
LABEL org.mobyproject.config='{"pid": "host", "readonly": true, "capabilities": ["CAP_SYS_ADMIN"]}'
