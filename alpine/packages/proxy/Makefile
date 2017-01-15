# Tag: 6075d4b9c5fe30e19581f1b7ef1813f3041cca32
GO_COMPILE=mobylinux/go-compile@sha256:badfd8a1730ab6e640682d0f95a8f9c51f3cd4b2e8db261fe1a1fd8c6f60bd6e

all: usr/bin/slirp-proxy sbin/proxy-vsockd

DEPS=$(wildcard *.go libproxy/*.go)

proxy: $(DEPS) ../vendor/manifest
	tar cf - $(DEPS) -C .. vendor | docker run --rm --net=none --log-driver=none -i $(GO_COMPILE) -o $@ | tar xf -

usr/bin/slirp-proxy: proxy
	mkdir -p usr/bin
	cp proxy $@

sbin/proxy-vsockd: proxy
	mkdir -p sbin
	cp proxy $@

clean:
	rm -rf proxy sbin usr

.DELETE_ON_ERROR:
