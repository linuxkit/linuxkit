FROM linuxkit/alpine:6fd232518678407a5ce4b31f7e21e07a883b4ba4@sha256:f5084a6b1716dd931749d1308529ee904f87fa74a5a9523f23046c4a1215424e AS mirror

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
LABEL org.mobyproject.config='{"net": "host","pid": "host", "ipc": "host", "readonly": true, "capabilities": ["CAP_SYS_ADMIN"]}'
