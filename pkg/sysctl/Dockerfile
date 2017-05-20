FROM linuxkit/alpine:5f6db26ab7bf6a9c452a612e236cc7495408132b@sha256:d009afc85d0b005daf51c8f3026aa552ab997dc47cab43915e9dc761accae086 AS mirror

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
