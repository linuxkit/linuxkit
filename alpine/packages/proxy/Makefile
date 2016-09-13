all: usr/bin/slirp-proxy sbin/proxy-vsockd

DEPS=Dockerfile $(wildcard *.go libproxy/*.go)

proxy: $(DEPS) ../vendor/manifest
	tar cf - $(DEPS) -C .. vendor | docker build -t proxy:build -
	docker run --rm --net=none proxy:build | tar xf -

usr/bin/slirp-proxy: proxy
	mkdir -p usr/bin
	cp proxy $@

sbin/proxy-vsockd: proxy
	mkdir -p sbin
	cp proxy $@

clean:
	rm -rf proxy sbin usr

.DELETE_ON_ERROR:
