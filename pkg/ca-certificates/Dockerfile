FROM linuxkit/alpine:8c02670fd2e5c28d03fd39db0bddc1379b3af6b2 as alpine

RUN apk add ca-certificates

FROM scratch
ENTRYPOINT []
WORKDIR /
COPY --from=alpine /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
