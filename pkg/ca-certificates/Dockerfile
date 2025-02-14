FROM linuxkit/alpine:35b33c6b03c40e51046c3b053dd131a68a26c37a as alpine

RUN apk add ca-certificates

FROM scratch
ENTRYPOINT []
WORKDIR /
COPY --from=alpine /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
