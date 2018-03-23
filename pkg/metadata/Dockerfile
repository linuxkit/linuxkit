FROM linuxkit/alpine:96ad1eb5ec262b4cd0eef574cdc0b225b502d9ee AS mirror

RUN apk add --no-cache go musl-dev linux-headers
ENV GOPATH=/go PATH=$PATH:/go/bin

COPY .  /go/src/metadata/
RUN go-compile.sh /go/src/metadata

RUN mkdir -p out/tmp out/var out/run out/dev out/etc out/etc/ssl/certs

FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=mirror /go/bin/metadata /usr/bin/metadata
COPY --from=mirror /out/ /
CMD ["/usr/bin/metadata"]
