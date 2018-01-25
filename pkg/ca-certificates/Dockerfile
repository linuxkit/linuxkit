FROM linuxkit/alpine:cba395fbc278daee841106801aba1e1bd7e0f2f7 as alpine

RUN apk add ca-certificates

FROM scratch
ENTRYPOINT []
WORKDIR /
COPY --from=alpine /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
