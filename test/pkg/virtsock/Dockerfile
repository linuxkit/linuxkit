FROM linuxkit/alpine:cba395fbc278daee841106801aba1e1bd7e0f2f7 AS mirror
RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
RUN apk add --no-cache --initdb -p /out \
    tini
RUN rm -rf /out/etc/apk /out/lib/apk /out/var/cache

FROM linuxkit/alpine:cba395fbc278daee841106801aba1e1bd7e0f2f7 AS build

RUN apk add --no-cache go musl-dev git make
ENV GOPATH=/go PATH=$PATH:/go/bin

ENV VIRTSOCK_COMMIT=3bfdf22e3b63a7d130ae5db41c2d76eaffa444d4
RUN mkdir -p $GOPATH/src/github.com/rneugeba && \
  cd $GOPATH/src/github.com/rneugeba && \
  git clone https://github.com/rneugeba/virtsock.git
WORKDIR $GOPATH/src/github.com/rneugeba/virtsock
RUN git checkout $VIRTSOCK_COMMIT
# Don't use go-compile.sh quite yet as the virtsock package is not yet lint free
RUN make build/virtsock_stress.linux && \
    cp -a build/virtsock_stress.linux /virtsock_stress

FROM scratch
COPY --from=mirror /out/ /
COPY --from=build virtsock_stress usr/bin/virtsock_stress
CMD ["/sbin/tini", "/usr/bin/virtsock_stress", "-s", "-v", "1"]
