FROM linuxkit/alpine:d497d5be5b78bfd78268f3466cc4080aee5d7389 as alpine

# The private keys of eight eMudhra intermediates chaining to
# emSign ECC Root CA - G3 are public. Alpine dropped the root in
# ca-certificates 20261020-r0; fetch it from upstream until linuxkit/alpine
# mirrors it, then go back to the plain mirror install.
RUN apk add --no-cache --repositories-file /etc/apk/repositories.upstream 'ca-certificates>=20261020-r0'

FROM scratch
ENTRYPOINT []
WORKDIR /
COPY --from=alpine /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
