FROM linuxkit/alpine:0c069d0fd7defddb6e03925fcd4915407db0c9e1 as alpine

RUN apk add ca-certificates

FROM scratch
ENTRYPOINT []
WORKDIR /
COPY --from=alpine /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
