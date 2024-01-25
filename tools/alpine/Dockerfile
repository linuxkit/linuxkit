ARG ALPINE_VERSION=3.19
FROM alpine:${ALPINE_VERSION} AS mirror

# update base image
RUN apk update && apk upgrade -a

# Copy Dockerfile so we can include it in the hash
COPY Dockerfile /Dockerfile
COPY packages* /tmp/

# mirror packages - both generic and repository specific ones
RUN cat /tmp/packages.$(uname -m) >> /tmp/packages && \
   mkdir -p /mirror/$(apk --print-arch) && \
   apk fetch --recursive -o /mirror/$(apk --print-arch) $(apk info; cat /tmp/packages)

# install abuild and sudo for signing
RUN apk add --no-cache abuild sudo

# install a new key into /etc/apk/keys
RUN abuild-keygen -a -i -n

# index the new repo
RUN apk index --rewrite-arch $(apk --print-arch) -o /mirror/$(apk --print-arch)/APKINDEX.unsigned.tar.gz /mirror/$(apk --print-arch)/*.apk

# sign the index
RUN cp /mirror/$(apk --print-arch)/APKINDEX.unsigned.tar.gz /mirror/$(apk --print-arch)/APKINDEX.tar.gz
RUN abuild-sign /mirror/$(apk --print-arch)/APKINDEX.tar.gz

# set this as our repo but keep a copy of the upstream for downstream use
RUN mv /etc/apk/repositories /etc/apk/repositories.upstream && echo "/mirror" > /etc/apk/repositories && apk update

# add Go validation tools
COPY go-compile.sh /go/bin/
RUN apk add --no-cache git go musl-dev
ENV GOPATH=/go PATH=$PATH:/go/bin
RUN go install golang.org/x/lint/golint@latest
RUN go install github.com/gordonklaus/ineffassign@latest

# Checkout and compile iucode-tool for Intel CPU microcode
# On non-x86_64 create a dummy file to copy below.
ENV IUCODE_REPO=https://gitlab.com/iucode-tool/iucode-tool
ENV IUCODE_COMMIT=v2.2
WORKDIR /
ADD iucode-tool.patch /
RUN set -e && \
    mkdir /iucode_tool && \
    if [ $(uname -m) = "x86_64" ]; then \
        apk add --no-cache automake autoconf argp-standalone git gcc make musl-dev patch && \
        git clone ${IUCODE_REPO} && \
        cd /iucode-tool && \
        git checkout ${IUCODE_COMMIT} && \
        patch -p 1 < /iucode-tool.patch && \
        ./autogen.sh && \
        ./configure && \
        make && \
        cp iucode_tool /iucode_tool; \
    fi

FROM alpine:${ALPINE_VERSION}

ARG TARGETARCH

COPY --from=mirror /etc/apk/repositories /etc/apk/repositories
COPY --from=mirror /etc/apk/repositories.upstream /etc/apk/repositories.upstream
COPY --from=mirror /etc/apk/keys /etc/apk/keys/
COPY --from=mirror /mirror /mirror/
COPY --from=mirror /go/bin /go/bin/
COPY --from=mirror /Dockerfile /Dockerfile
COPY --from=mirror /iucode_tool /usr/bin/

RUN apk update && apk upgrade -a

RUN echo Dockerfile /lib/apk/db/installed $(find /mirror -name '*.apk' -type f) $(find /go/bin -type f) | xargs cat | sha1sum | sed 's/ .*//' | sed 's/$/-'"${TARGETARCH}"'/' > /etc/alpine-hash-arch

ENV GOPATH=/go PATH=$PATH:/go/bin
