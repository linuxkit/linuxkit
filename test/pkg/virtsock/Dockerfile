FROM linuxkit/alpine:630ee558e4869672fae230c78364e367b8ea67a9 AS mirror
RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
RUN apk add --no-cache --initdb -p /out \
    tini
RUN rm -rf /out/etc/apk /out/lib/apk /out/var/cache

FROM linuxkit/go-compile:4513068d9a7e919e4ec42e2d7ee879ff5b95b7f5@sha256:bdfadbe3e4ec699ca45b67453662321ec270f2d1a1dbdbf09625776d3ebd68c5 AS build

ENV VIRTSOCK_COMMIT=3bfdf22e3b63a7d130ae5db41c2d76eaffa444d4
RUN mkdir -p $GOPATH/src/github.com/rneugeba && \
  cd $GOPATH/src/github.com/rneugeba && \
  git clone https://github.com/rneugeba/virtsock.git
WORKDIR $GOPATH/src/github.com/rneugeba/virtsock
RUN git checkout $VIRTSOCK_COMMIT
RUN make build/virtsock_stress.linux
RUN cp -a build/virtsock_stress.linux /virtsock_stress

FROM scratch
COPY --from=mirror /out/ /
COPY --from=build virtsock_stress bin/virtsock_stress
CMD ["/sbin/tini", "/bin/virtsock_stress", "-s", "-v", "1"]
