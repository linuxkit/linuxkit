FROM linuxkit/alpine:6336329f15b4166514782eaa555cf0ffd35c519c@sha256:f6c2ce92910b1d6e4e5557850a554f4a3ae9f66c1e89ad86a24d6c6e550f165e as alpine

RUN apk add ca-certificates

FROM scratch
ENTRYPOINT []
WORKDIR /
COPY --from=alpine /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
