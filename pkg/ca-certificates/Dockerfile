FROM linuxkit/alpine:33063834cf72d563cd8703467836aaa2f2b5a300 as alpine

RUN apk add ca-certificates

FROM scratch
ENTRYPOINT []
WORKDIR /
COPY --from=alpine /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
