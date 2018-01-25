FROM linuxkit/alpine:cba395fbc278daee841106801aba1e1bd7e0f2f7 AS build

ENV TROUSERS_COMMIT de57f069ef2297d6a6b3a0353e217a5a2f66e444
ENV TPM_TOOLS_COMMIT bdf9f1bc8f63cd6fc370c2deb58d03ac55079e84

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
LABEL org.mobyproject.config='{"pid": "host", "net":"host", "binds": ["/dev:/dev"], "capabilities": ["all"]}'
