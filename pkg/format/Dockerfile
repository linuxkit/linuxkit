FROM linuxkit/alpine:c5021a113803f7608e3908014d316c3490183d0b@sha256:4c7a80173c71ca5019dc56f40442b3a8345f141dd46593bd1fe6d130294d07cb AS mirror

FROM alpine:3.5@sha256:dfbd4a3a8ebca874ebd2474f044a0b33600d4523d03b0df76e5c5986cb02d7e8
COPY --from=mirror /etc/apk/repositories /etc/apk/repositories
COPY --from=mirror /etc/apk/keys /etc/apk/keys/
COPY --from=mirror /mirror /mirror/

RUN apk add \
    e2fsprogs \
    e2fsprogs-extra \
    jq \
    sfdisk \
    && true

RUN rm -rf /mirror /etc/apk/repositories /etc/apk/keys

COPY . ./
CMD ["/bin/sh", "/format.sh"]
