all: usr/bin/slirp-proxy sbin/proxy-vsockd

DEPS=Dockerfile $(wildcard *.go libproxy/*.go)

# Tag: 02a04b2dc262195773b29125daebca0eebd97845
LINT_IMAGE=mobylinux/alpine-go-lint@sha256:3009197c25294d87a7e81a53493666de47ed2b55a5bd283eabad4b876ea64559

proxy: $(DEPS) ../vendor/manifest lint
	BUILD=$$( tar cf - $(DEPS) -C .. vendor | docker build -q - ) && \
	[ -n "$$BUILD" ] && \
	echo "Built $$BUILD" && \
	docker run --rm --net=none $$BUILD | tar xf -

usr/bin/slirp-proxy: proxy
	mkdir -p usr/bin
	cp proxy $@

sbin/proxy-vsockd: proxy
	mkdir -p sbin
	cp proxy $@

clean:
	rm -rf proxy sbin usr

lint:
	docker run --rm -v $(PWD):/src $(LINT_IMAGE)

.DELETE_ON_ERROR:
