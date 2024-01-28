FROM linuxkit/alpine:146f540f25cd92ec8ff0c5b0c98342a9a95e479e AS mirror
RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
RUN apk add --no-cache --initdb -p /out \
    tini
RUN rm -rf /out/etc/apk /out/lib/apk /out/var/cache

FROM linuxkit/alpine:146f540f25cd92ec8ff0c5b0c98342a9a95e479e AS build

RUN apk add --no-cache go musl-dev git make
ENV GOPATH=/go PATH=$PATH:/go/bin GO111MODULE=off

ENV VIRTSOCK_COMMIT=f1e32d3189e0dbb81c0e752a4e214617487eb41f
RUN mkdir -p $GOPATH/src/github.com/linuxkit && \
  cd $GOPATH/src/github.com/linuxkit && \
  git clone https://github.com/linuxkit/virtsock.git
WORKDIR $GOPATH/src/github.com/linuxkit/virtsock
RUN git checkout $VIRTSOCK_COMMIT
# Don't use go-compile.sh quite yet as the virtsock package is not yet lint free
RUN make bin/sock_stress.linux && \
    cp -a bin/sock_stress.linux /sock_stress

FROM scratch
COPY --from=mirror /out/ /
COPY --from=build sock_stress usr/bin/sock_stress
CMD ["/sbin/tini", "/usr/bin/sock_stress", "-s", "-v", "1"]
