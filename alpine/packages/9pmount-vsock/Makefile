DEPS=Dockerfile $(wildcard *.c *.h)

9pmount-vsock: $(DEPS)
	mkdir -p sbin
	BUILD=$$( tar cf - $(DEPS) | docker build -q - ) && \
	[ -n "$$BUILD" ] && \
	echo "Built $$BUILD" && \
	docker run --rm --net=none $$BUILD | tar xf - -C sbin


clean:
	rm -rf sbin
