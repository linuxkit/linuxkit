FROM alpine:3.5 as alpine

RUN apk update && apk upgrade -a && apk add --no-cache ca-certificates

FROM scratch
ENTRYPOINT []
WORKDIR /
COPY --from=alpine /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
