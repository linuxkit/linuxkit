FROM linuxkit/alpine:28254e4530703db4caa6b0199a025c30a987dfa1 AS build

RUN apk add --no-cache \
    musl-dev git build-base openssl-dev linux-headers autoconf automake \
    libtool bison flex file
ENV WPA_2_6=a68d8f6a0fd493dca8b7049ea7a6a51b7f95cac0
ENV LIBNL3_4_0RC1=e01b9df629e2f4f833fdc4fe0bda460bb738d136

RUN mkdir -p /usr/src
RUN git clone -n git://w1.fi/srv/git/hostap.git /usr/src/hostap && \
    cd /usr/src/hostap && \
    git checkout $WPA_2_6 && \
    git clone -n git://git.infradead.org/users/tgr/libnl.git /usr/src/libnl && \
    cd /usr/src/libnl && \
    git checkout $LIBNL3_4_0RC1

COPY config /usr/src/hostap/wpa_supplicant/.config

RUN cd /usr/src/libnl && \
    export CFLAGS="$CFLAGS -D__GLIBC__" && \
    autoreconf --install && \
    ./configure --prefix=/usr --enable-cli=no && \
    make && \
    make install && \
    cd /usr/src/hostap/wpa_supplicant && \
    make

FROM linuxkit/alpine:28254e4530703db4caa6b0199a025c30a987dfa1 AS mirror

RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
RUN apk add --initdb -p /out alpine-baselayout openssl

# Remove apk residuals. We have a read-only rootfs, so apk is of no use.
RUN rm -rf /out/etc/apk /out/lib/apk /out/var/cache

FROM scratch
COPY --from=mirror /out/ /
COPY --from=build /usr/lib/libnl-* /usr/lib/
COPY --from=build /usr/src/hostap/wpa_supplicant/wpa_supplicant /sbin/wpa_supplicant
COPY --from=build /usr/src/hostap/wpa_supplicant/wpa_cli /sbin/wpa_cli
CMD []
LABEL org.mobyproject.config='{"capabilities": ["CAP_NET_ADMIN", "CAP_NET_RAW"]}'
