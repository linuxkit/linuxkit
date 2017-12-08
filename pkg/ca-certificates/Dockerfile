FROM linuxkit/alpine:d307c8a386fa3f32cddda9409b9687e191cdd6f1 as alpine

RUN apk add ca-certificates

FROM scratch
ENTRYPOINT []
WORKDIR /
COPY --from=alpine /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
