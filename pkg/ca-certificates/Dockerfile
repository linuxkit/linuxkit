FROM linuxkit/alpine:86cd4f51b49fb9a078b50201d892a3c7973d48ec as alpine

RUN apk add ca-certificates

FROM scratch
ENTRYPOINT []
WORKDIR /
COPY --from=alpine /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
