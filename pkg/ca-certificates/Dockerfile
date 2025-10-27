FROM linuxkit/alpine:8382c70bfe6b581d2c6b3032ca1081ef2ef76e1f as alpine

RUN apk add ca-certificates

FROM scratch
ENTRYPOINT []
WORKDIR /
COPY --from=alpine /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
