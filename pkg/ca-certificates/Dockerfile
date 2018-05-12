FROM linuxkit/alpine:1b05307ae8152e3d38f79e297b0632697a30c65c as alpine

RUN apk add ca-certificates

FROM scratch
ENTRYPOINT []
WORKDIR /
COPY --from=alpine /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
