all: usr/bin/slirp-proxy sbin/proxy-vsockd

DEPS=Dockerfile $(wildcard *.go libproxy/*.go)

proxy: $(DEPS) ../vendor/manifest
	BUILD=$$( tar cf - $(DEPS) -C .. vendor | docker build -q - ) && [ -n "$$BUILD" ] && \
	docker run --rm --net=none $$BUILD | tar xf - && \
	docker rmi --no-prune $$BUILD

usr/bin/slirp-proxy: proxy
	mkdir -p usr/bin
	cp proxy $@

sbin/proxy-vsockd: proxy
	mkdir -p sbin
	cp proxy $@

clean:
	rm -rf proxy sbin usr

.DELETE_ON_ERROR:
