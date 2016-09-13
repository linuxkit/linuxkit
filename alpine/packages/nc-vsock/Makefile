DEPS=Dockerfile $(wildcard *.c *.h)

usr/bin/nc-vsock: $(DEPS)
	mkdir -p usr/bin
	tar cf - $(DEPS) | docker build -t nc-vsock:build -
	docker run --rm --net=none nc-vsock:build | tar xf - -C usr/bin

clean:
	rm -rf usr
