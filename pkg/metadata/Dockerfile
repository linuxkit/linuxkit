FROM linuxkit/alpine:451603daf499e3a40308dbf5571dcffed2343ffa AS mirror

RUN apk add --no-cache go musl-dev
ENV GOPATH=/go PATH=$PATH:/go/bin

COPY .  /go/src/metadata/
RUN go-compile.sh /go/src/metadata

FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=mirror /go/bin/metadata /usr/bin/metadata
CMD ["/usr/bin/metadata"]
LABEL org.mobyproject.config='{"binds": ["/dev:/dev", "/var:/var", "/tmp/etc/resolv.conf:/etc/resolv.conf"], "capabilities": ["CAP_SYS_ADMIN"]}'
