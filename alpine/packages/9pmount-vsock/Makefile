DEPS=Dockerfile $(wildcard *.c *.h)

9pmount-vsock: $(DEPS)
	mkdir -p sbin
	tar cf - $(DEPS) | docker build -t 9pmount-vsock:build -
	docker run --rm --net=none 9pmount-vsock:build | tar xf - -C sbin

clean:
	rm -rf sbin
