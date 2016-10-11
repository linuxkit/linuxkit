all: usr/bin/diagnostics-server

DEPS=Dockerfile $(wildcard *.go)

usr/bin/diagnostics-server: $(DEPS) ../vendor/manifest
	BUILD=$$( tar cf - $(DEPS) -C .. vendor | docker build -q - ) && \
	[ -n "$$BUILD" ] && \
	echo "Built $$BUILD" && \
	docker run --rm --net=none $$BUILD | tar xf - -C usr/bin

clean:
	rm -f usr/bin/diagnostics-server

.DELETE_ON_ERROR:
