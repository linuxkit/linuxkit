# Use sha256 here to get a fixed version
FROM alpine:edge
ENTRYPOINT []
CMD []
WORKDIR /
COPY init /
COPY etc etc/

RUN ip a
RUN apk update
RUN apk add --no-cache attr openssl
RUN rm -rf /mirror /etc/apk/repositories /etc/apk/keys
