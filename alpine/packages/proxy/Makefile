GO_COMPILE=mobylinux/go-compile:1ccd40acabb46f43593a960f5d3e0a42df53fb90@sha256:208c795a9b6572df09d9bf52ff3cfa97b9a07d109811ec9d28453a22c53f9d17

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
