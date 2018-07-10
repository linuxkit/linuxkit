FROM linuxkit/alpine:daed76b8f1d28cdeeee215a95b9671c682a405dc as alpine

RUN apk add ca-certificates

FROM scratch
ENTRYPOINT []
WORKDIR /
COPY --from=alpine /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
