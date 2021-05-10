FROM alpine:3.13 AS mirror

# update base image
RUN apk update && apk upgrade -a

# Copy Dockerfile so we can include it in the hash
COPY Dockerfile /Dockerfile
COPY packages* /tmp/

# mirror packages - both generic and repository specific ones
RUN cat /tmp/packages.$(uname -m) >> /tmp/packages && \
   mkdir -p /mirror/$(apk --print-arch) && \
   apk fetch --recursive -o /mirror/$(apk --print-arch) $(apk info; cat /tmp/packages)

# these are the repository-specific ones, if they exist
RUN for repopkgs in /tmp/packages.repo.*; do \
        repo=${repopkgs##*repo.} && archspecific=/tmp/packages.$(uname -m).repo.${repo} && mergedfile=/tmp/packages.merged.${repo} && repofile=/tmp/repositories.${repo} && \
	cachedir=/tmp/cache/${repo} && \
	mkdir -p ${cachedir} && \
        cp ${repopkgs} ${mergedfile} && \
        if [ -f ${archspecific} ]; then cat ${archspecific} >> ${mergedfile}; fi && \
        sed "s#alpine/[^\/]*/#alpine/${repo}/#g" /etc/apk/repositories > ${repofile} && \
        apk update --repositories-file=${repofile} --cache-dir ${cachedir} && \
        apk fetch --repositories-file=${repofile} --cache-dir ${cachedir} --recursive -o /mirror/$(apk --print-arch) $(cat ${mergedfile}); done

# we CANNOT allow musl-dev or musl > 1.2.2-r0, which ships with alpine:3.13, because 1.2.2-r2, which ships with alpine:edge / alpine:3.14
#   uses the new faccessat2 system call which gives errors, see https://wiki.alpinelinux.org/wiki/Draft_Release_Notes_for_Alpine_3.14.0#faccessat2
RUN target="1.2.2-r0" && \
    verlte() { [  "$1" = $(printf '%s\n%s' "$1" "$2" | sort -V | head -n1) ] ; } && \
    for file in /mirror/$(apk --print-arch)/musl-*.apk; do \
    version=$(tar -xf ${file}  -O .PKGINFO | awk '$1 == "pkgver" {print $3}') && \
    if ! verlte ${version} ${target} ; then echo "removing ${file}" && rm ${file}; fi; done

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
RUN go get -u golang.org/x/lint/golint
RUN go get -u github.com/gordonklaus/ineffassign
RUN go get -u github.com/LK4D4/vndr

# checkout and compile containerd
# Update `FROM` in `pkg/containerd/Dockerfile`, `pkg/init/Dockerfile` and
# `test/pkg/containerd/Dockerfile` when changing this.
# when building, note that containerd does not use go modules in the below commit,
# while go1.16 defaults to using it, so must disable with GO111MODULE=off
ENV CONTAINERD_REPO=https://github.com/containerd/containerd.git
ENV CONTAINERD_COMMIT=v1.4.4
RUN mkdir -p $GOPATH/src/github.com/containerd && \
  cd $GOPATH/src/github.com/containerd && \
  git clone https://github.com/containerd/containerd.git && \
  cd $GOPATH/src/github.com/containerd/containerd && \
  git checkout $CONTAINERD_COMMIT
RUN apk add --no-cache btrfs-progs-dev gcc libc-dev linux-headers make libseccomp-dev
RUN cd $GOPATH/src/github.com/containerd/containerd && \
  GO111MODULE=off make binaries EXTRA_FLAGS="-buildmode pie" EXTRA_LDFLAGS='-extldflags "-fno-PIC -static"' BUILDTAGS="static_build no_devmapper"

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

FROM alpine:3.13

ARG TARGETARCH

COPY --from=mirror /etc/apk/repositories /etc/apk/repositories
COPY --from=mirror /etc/apk/repositories.upstream /etc/apk/repositories.upstream
COPY --from=mirror /etc/apk/keys /etc/apk/keys/
COPY --from=mirror /mirror /mirror/
COPY --from=mirror /go/bin /go/bin/
COPY --from=mirror /Dockerfile /Dockerfile
COPY --from=mirror /go/src/github.com/containerd/containerd /go/src/github.com/containerd/containerd/
COPY --from=mirror /iucode_tool /usr/bin/

RUN apk update && apk upgrade -a

RUN echo Dockerfile /lib/apk/db/installed $(find /mirror -name '*.apk' -type f) $(find /go/bin -type f) | xargs cat | sha1sum | sed 's/ .*//' | sed 's/$/-'"${TARGETARCH}"'/' > /etc/alpine-hash-arch

ENV GOPATH=/go PATH=$PATH:/go/bin
