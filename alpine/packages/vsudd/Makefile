all: vsudd

DEPS=Dockerfile $(wildcard *.go)

vsudd: $(DEPS) ../vendor/manifest
	mkdir -p sbin
	BUILD=$$( tar cf - $(DEPS) -C .. vendor | docker build -q - ) && \
	[ -n "$$BUILD" ] && \
	echo "Built $$BUILD" && \
	docker run --rm --net=none $$BUILD | tar xf - -C sbin

clean:
	rm -rf sbin

.DELETE_ON_ERROR:
