FROM linuxkit/alpine:c608d404c1cb36cce0c7d9303e30b52c9d81ccf0 as alpine

RUN apk add ca-certificates

FROM scratch
ENTRYPOINT []
WORKDIR /
COPY --from=alpine /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
