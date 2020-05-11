FROM linuxkit/alpine:e2391e0b164c57db9f6c4ae110ee84f766edc430 as alpine

RUN apk add ca-certificates

FROM scratch
ENTRYPOINT []
WORKDIR /
COPY --from=alpine /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
