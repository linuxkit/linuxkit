FROM linuxkit/alpine:6264e5b39af8eb1da7ffa4c05a7ccc597da01197 as alpine

RUN apk add ca-certificates

FROM scratch
ENTRYPOINT []
WORKDIR /
COPY --from=alpine /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
