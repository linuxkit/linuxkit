FROM linuxkit/alpine:316c3f9d85c21fdd8bc7479e81d290f85bf60eb0 as alpine

RUN apk add ca-certificates

FROM scratch
ENTRYPOINT []
WORKDIR /
COPY --from=alpine /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
