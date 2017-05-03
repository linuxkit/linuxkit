FROM alpine:edge AS mirror

# update base image
RUN apk update && apk upgrade -a

COPY packages /tmp/

# get full set of dependent packages by installing
RUN cat /tmp/packages | xargs apk add --no-cache
RUN apk info -v | sed 's/$/.apk/g' > /tmp/exact-packages

# install rsync, abuild for mirroring, gcc dep for abuild-keygen
RUN apk add --no-cache rsync abuild gcc

# mirror packages
RUN mkdir -p /mirror/$(uname -m) && \
  rsync --files-from=/tmp/exact-packages rsync://rsync.alpinelinux.org/alpine/edge/main/$(uname -m)/ /mirror/$(uname -m)/

# install a new key into /etc/apk/keys
RUN abuild-keygen -a -i -n

# index the new repo
RUN apk index -o /mirror/$(uname -m)/APKINDEX.unsigned.tar.gz /mirror/$(uname -m)/*.apk

# sign the index
RUN cp /mirror/$(uname -m)/APKINDEX.unsigned.tar.gz /mirror/$(uname -m)/APKINDEX.tar.gz
RUN abuild-sign /mirror/$(uname -m)/APKINDEX.tar.gz

# set this as our repo
RUN echo "/mirror" > /etc/apk/repositories && apk update

FROM alpine:edge

# update base image
RUN apk update && apk upgrade -a

COPY --from=mirror /etc/apk/repositories /etc/apk/repositories
COPY --from=mirror /etc/apk/keys /etc/apk/keys/
COPY --from=mirror /mirror /mirror/
