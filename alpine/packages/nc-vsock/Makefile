DEPS=Dockerfile $(wildcard *.c *.h)

usr/bin/nc-vsock: $(DEPS)
	mkdir -p usr/bin
	BUILD=$$( tar cf - $(DEPS) | docker build -q - ) && \
	[ -n "$$BUILD" ] && \
	echo "Built $$BUILD" && \
	docker run --rm --net=none $$BUILD | tar xf - -C usr/bin

clean:
	rm -rf usr
