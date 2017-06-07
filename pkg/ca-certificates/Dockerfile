FROM linuxkit/containerd:5749f2e9e65395cc6635229e8da0e0d484320ddf as alpine

RUN apk add ca-certificates

FROM scratch
ENTRYPOINT []
WORKDIR /
COPY --from=alpine /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
