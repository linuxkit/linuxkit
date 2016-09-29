all: vsudd

DEPS=Dockerfile $(wildcard *.go)

vsudd: $(DEPS) ../vendor/manifest
	mkdir -p sbin
	BUILD=$$( tar cf - $(DEPS) -C .. vendor | docker build -q - ) && [ -n "$$BUILD" ] && \
	docker run --rm --net=none $$BUILD | tar xf - -C sbin && \
	docker rmi --no-prune $$BUILD

clean:
	rm -rf sbin

.DELETE_ON_ERROR:
