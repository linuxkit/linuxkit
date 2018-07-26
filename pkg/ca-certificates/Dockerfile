FROM linuxkit/alpine:3683c9a66cd4da40bd7d6c7da599b2dcd738b559 as alpine

RUN apk add ca-certificates

FROM scratch
ENTRYPOINT []
WORKDIR /
COPY --from=alpine /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
