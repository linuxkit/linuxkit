FROM linuxkit/alpine:bc528cf9d4065d2e09aa44ff76909b94cfe8d867 as alpine

RUN apk add ca-certificates

FROM scratch
ENTRYPOINT []
WORKDIR /
COPY --from=alpine /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
