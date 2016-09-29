all: usr/bin/diagnostics-server

DEPS=Dockerfile $(wildcard *.go)

usr/bin/diagnostics-server: $(DEPS) ../vendor/manifest
	BUILD=$$( tar cf - $(DEPS) -C .. vendor | docker build -q - ) && [ -n "$$BUILD" ] && \
	docker run --rm --net=none $$BUILD | tar xf - -C usr/bin && \
	docker rmi --no-prune $$BUILD

clean:
	rm -f usr/bin/diagnostics-server

.DELETE_ON_ERROR:
