FROM alpine:edge AS mirror

# update base image
RUN apk update && apk upgrade -a

COPY packages /tmp/

# mirror packages
RUN mkdir -p /mirror/$(uname -m) && \
   apk fetch --recursive -o /mirror/$(uname -m) $(apk info; cat /tmp/packages)

# install abuild for signing
RUN apk add --no-cache abuild

# install a new key into /etc/apk/keys
RUN abuild-keygen -a -i -n

# index the new repo
RUN apk index --rewrite-arch $(uname -m) -o /mirror/$(uname -m)/APKINDEX.unsigned.tar.gz /mirror/$(uname -m)/*.apk

# sign the index
RUN cp /mirror/$(uname -m)/APKINDEX.unsigned.tar.gz /mirror/$(uname -m)/APKINDEX.tar.gz
RUN abuild-sign /mirror/$(uname -m)/APKINDEX.tar.gz

# set this as our repo
RUN echo "/mirror" > /etc/apk/repositories && apk update

FROM alpine:edge

COPY --from=mirror /etc/apk/repositories /etc/apk/repositories
COPY --from=mirror /etc/apk/keys /etc/apk/keys/
COPY --from=mirror /mirror /mirror/

RUN apk update && apk upgrade -a
