FROM linuxkit/alpine:96ad1eb5ec262b4cd0eef574cdc0b225b502d9ee as alpine

RUN apk add ca-certificates

FROM scratch
ENTRYPOINT []
WORKDIR /
COPY --from=alpine /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
