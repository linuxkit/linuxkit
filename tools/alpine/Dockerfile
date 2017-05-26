FROM alpine:3.6 AS mirror

# update base image
RUN apk update && apk upgrade -a

# Copy Dockerfile so we can include it in the hash
COPY Dockerfile /Dockerfile
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

# fetch OVMF for qemu EFI boot (this is not added as a package)
RUN apk add -X http://dl-cdn.alpinelinux.org/alpine/edge/community ovmf

# set this as our repo
RUN echo "/mirror" > /etc/apk/repositories && apk update

# add Go validation tools
COPY go-compile.sh /go/bin/
RUN apk add --no-cache git go musl-dev
ENV GOPATH=/go PATH=$PATH:/go/bin
RUN go get -u github.com/golang/lint/golint
RUN go get -u github.com/gordonklaus/ineffassign
RUN go get -u github.com/LK4D4/vndr

FROM koalaman/shellcheck:v0.4.6@sha256:191b61e5f436fc51f22faaf2f4e0f77799f75977c7210377dd73a1a0f99ef8bd AS shellcheck


FROM alpine:3.6

COPY --from=mirror /etc/apk/repositories /etc/apk/repositories
COPY --from=mirror /etc/apk/keys /etc/apk/keys/
COPY --from=mirror /mirror /mirror/
COPY --from=mirror /go/bin /go/bin/
COPY --from=mirror /usr/share/ovmf/bios.bin /usr/share/ovmf/bios.bin
COPY --from=mirror /Dockerfile /Dockerfile

COPY --from=shellcheck /usr/local/bin/shellcheck /usr/local/bin/shellcheck
COPY --from=shellcheck /usr/local/lib/ /usr/local/lib/

RUN apk update && apk upgrade -a

ARG CONTAINERD_COMMIT=c215531a8f63a98a69134e804fea4ee6d354bb90
