GO_COMPILE=mobylinux/go-compile:d2d25ac665b5148ad356d0eab3ff3762a68c633d@sha256:aab55d0c317460850e66a07dd94139cc11ea9e1c0bee88716a6a8c768740885f

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
