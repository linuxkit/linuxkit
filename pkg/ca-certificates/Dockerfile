FROM linuxkit/alpine:9bcf61f605ef0ce36cc94d59b8eac307862de6e1 as alpine

RUN apk add ca-certificates

FROM scratch
ENTRYPOINT []
WORKDIR /
COPY --from=alpine /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
