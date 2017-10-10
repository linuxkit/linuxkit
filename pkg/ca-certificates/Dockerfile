FROM linuxkit/alpine:ad35b6ddbc70faa07e59a9d7dee7707c08122e8d as alpine

RUN apk add ca-certificates

FROM scratch
ENTRYPOINT []
WORKDIR /
COPY --from=alpine /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
