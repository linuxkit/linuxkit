FROM linuxkit/alpine:630ee558e4869672fae230c78364e367b8ea67a9 as alpine

RUN apk add ca-certificates

FROM scratch
ENTRYPOINT []
WORKDIR /
COPY --from=alpine /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
