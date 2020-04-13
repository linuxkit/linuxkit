FROM linuxkit/alpine:bc528cf9d4065d2e09aa44ff76909b94cfe8d867 AS mirror

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
