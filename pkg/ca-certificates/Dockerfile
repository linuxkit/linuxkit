FROM linuxkit/alpine:518c2ed0f398c5508969ac5e033607201fb419ed as alpine

RUN apk add ca-certificates

FROM scratch
ENTRYPOINT []
WORKDIR /
COPY --from=alpine /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
