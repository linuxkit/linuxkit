FROM linuxkit/alpine:146f540f25cd92ec8ff0c5b0c98342a9a95e479e AS build

ENV TROUSERS_COMMIT 94144b0a1dcef6e31845d6c319e9bd7357208eb9
ENV TPM_TOOLS_COMMIT bf43837575c5f7d31865562dce7778eae970052e

RUN apk add --no-cache --initdb \
    automake \
    autoconf \
    gettext \
    gettext-dev \
    git \
    pkgconfig \
    libtool \
    libc-dev \
    linux-headers \
    gcc \
    make \
    openssl-dev \
    util-linux \
    && true

RUN mkdir -p /usr/src
COPY src/glibc_stubs/ /usr/src/glibc_stubs
WORKDIR /usr/src/glibc_stubs
RUN make && make install

RUN git clone https://git.code.sf.net/p/trousers/trousers /usr/src/trousers-trousers && cd /usr/src/trousers-trousers && git checkout $TROUSERS_COMMIT
RUN git clone https://git.code.sf.net/p/trousers/tpm-tools /usr/src/trousers-tpm-tools && cd /usr/src/trousers-tpm-tools && git checkout $TPM_TOOLS_COMMIT
WORKDIR /usr/src/trousers-trousers
RUN sh bootstrap.sh && \
    ./configure --prefix=/ --sysconfdir=/etc LDFLAGS="-L/out/lib/ -lgetpwent_r" && \
    make && \
    make install prefix=/out
WORKDIR /usr/src/trousers-tpm-tools

RUN sh bootstrap.sh && \
    ./configure --prefix=/out CFLAGS="-I/out/include" LDFLAGS="-L/out/lib/ -lgetpwent_r" && \
    make && \
    make install

COPY src/savedeps/ /usr/src/savedeps
RUN /usr/src/savedeps/savedeps.sh /out /out

RUN mkdir -p /out/var/lib/tpm

# we need busybox to run chmod, chown, touch, etc.
RUN mkdir -p /out/bin && cp /bin/busybox /out/bin/busybox && ln -s /bin/busybox /out/bin/sh



FROM scratch
WORKDIR /
COPY --from=build /out /
COPY etc/ /etc
# set up the appropriate groups and perms
RUN busybox chmod 0644 /etc/passwd /etc/group && \
    busybox chmod 0640 /etc/shadow && \
    busybox touch /etc/tcsd.conf && \
    busybox chmod 0600 /etc/tcsd.conf && \
    busybox chown -R tss.tss /var/lib/tpm/ /etc/tcsd.conf && \
    busybox rm /bin/busybox /bin/sh

CMD ["/sbin/tcsd","-f"]
