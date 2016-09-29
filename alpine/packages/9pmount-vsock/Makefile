DEPS=Dockerfile $(wildcard *.c *.h)

9pmount-vsock: $(DEPS)
	mkdir -p sbin
	BUILD=$$( tar cf - $(DEPS) | docker build -q - ) && [ -n "$$BUILD" ] && \
	docker run --rm --net=none $$BUILD | tar xf - -C sbin && \
	docker rmi --no-prune $$BUILD

clean:
	rm -rf sbin
