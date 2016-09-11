all: vsudd

DEPS=Dockerfile $(wildcard *.go)

vsudd: $(DEPS) ../vendor/manifest
	mkdir -p sbin
	tar cf - $(DEPS) -C .. vendor | docker build -t vsudd:build -
	docker run --rm --net=none vsudd:build | tar xf - -C sbin

clean:
	rm -rf sbin

.DELETE_ON_ERROR:
