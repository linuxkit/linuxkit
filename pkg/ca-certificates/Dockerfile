FROM linuxkit/alpine:a720d2b1b83556653abd52d61faa0fe020c2b161 as alpine

RUN apk add ca-certificates

FROM scratch
ENTRYPOINT []
WORKDIR /
COPY --from=alpine /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
