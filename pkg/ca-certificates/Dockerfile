FROM linuxkit/alpine:275ef778d89133ebec5e80d82c664c3549f919d4 as alpine

RUN apk add ca-certificates

FROM scratch
ENTRYPOINT []
WORKDIR /
COPY --from=alpine /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
