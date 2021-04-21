FROM linuxkit/alpine:f0af677747790eda63ee5e9b37b9002b965f48dc as alpine

RUN apk add ca-certificates

FROM scratch
ENTRYPOINT []
WORKDIR /
COPY --from=alpine /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
