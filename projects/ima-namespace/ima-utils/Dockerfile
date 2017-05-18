FROM alpine:edge as utils
RUN apk add --no-cache attr openssl

FROM scratch
ENTRYPOINT []
CMD []
WORKDIR /
COPY --from=utils /usr/bin/openssl /usr/bin/setfattr /usr/bin/
COPY --from=utils /lib/libattr.so* /lib/libssl.so* /lib/libcrypto.so* /lib/
