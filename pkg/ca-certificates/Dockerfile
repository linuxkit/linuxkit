FROM linuxkit/alpine:146f540f25cd92ec8ff0c5b0c98342a9a95e479e as alpine

RUN apk add ca-certificates

FROM scratch
ENTRYPOINT []
WORKDIR /
COPY --from=alpine /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
