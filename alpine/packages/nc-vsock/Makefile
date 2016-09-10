DEPS=Dockerfile $(wildcard *.c *.h)

nc-vsock: $(DEPS)
	tar cf - $(DEPS) | docker build -t nc-vsock:build -
	docker run --rm --net=none nc-vsock:build | tar xf -

clean:
	rm -f nc-vsock
