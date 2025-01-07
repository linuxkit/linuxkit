FROM linuxkit/alpine:3fb2ec0c94ed30171bb04100f568bba2ebb648a9 as alpine

RUN apk add ca-certificates

FROM scratch
ENTRYPOINT []
WORKDIR /
COPY --from=alpine /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
