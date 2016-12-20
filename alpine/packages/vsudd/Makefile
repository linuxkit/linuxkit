all: vsudd

DEPS=Dockerfile $(wildcard *.go)

# Tag: 02a04b2dc262195773b29125daebca0eebd97845
LINT_IMAGE=mobylinux/alpine-go-lint@sha256:3009197c25294d87a7e81a53493666de47ed2b55a5bd283eabad4b876ea64559

vsudd: $(DEPS) ../vendor/manifest lint
	mkdir -p sbin
	BUILD=$$( tar cf - $(DEPS) -C .. vendor | docker build -q - ) && \
	[ -n "$$BUILD" ] && \
	echo "Built $$BUILD" && \
	docker run --rm --net=none $$BUILD | tar xf - -C sbin

clean:
	rm -rf sbin

lint:
	docker run --rm -v $(PWD):/src $(LINT_IMAGE)

.DELETE_ON_ERROR:
