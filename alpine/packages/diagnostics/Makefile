all: usr/bin/diagnostics-server

DEPS=Dockerfile $(wildcard *.go)

usr/bin/diagnostics-server: $(DEPS) ../vendor/manifest
	tar cf - $(DEPS) -C .. vendor | docker build -t diagnostics-server:build -
	docker run --rm --net=none diagnostics-server:build | tar xf - -C usr/bin

clean:
	rm -f usr/bin/diagnostics-server

.DELETE_ON_ERROR:
