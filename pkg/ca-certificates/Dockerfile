FROM linuxkit/alpine:f0169b60fb260d74025496ae6fd93213fecaba8f as alpine

RUN apk add ca-certificates

FROM scratch
ENTRYPOINT []
WORKDIR /
COPY --from=alpine /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
