FROM linuxkit/alpine:e9313150d2cf07ad92b0adb86799dedd9a0476e4 as alpine

RUN apk add ca-certificates

FROM scratch
ENTRYPOINT []
WORKDIR /
COPY --from=alpine /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
