DEPS=Dockerfile $(wildcard *.c *.h)

9pmount-vsock: $(DEPS)
	tar cf - $(DEPS) | docker build -t 9pmount-vsock:build -
	docker run --rm --net=none 9pmount-vsock:build | tar xf -

clean:
	rm -f 9pmount-vsock
