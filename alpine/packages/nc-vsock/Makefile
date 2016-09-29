DEPS=Dockerfile $(wildcard *.c *.h)

usr/bin/nc-vsock: $(DEPS)
	mkdir -p usr/bin
	BUILD=$$( tar cf - $(DEPS) | docker build -q - ) && [ -n "$$BUILD" ] && \
	docker run --rm --net=none $$BUILD | tar xf - -C usr/bin && \
	docker rmi --no-prune $$BUILD

clean:
	rm -rf usr
