DEPS=Dockerfile $(wildcard *.c *.h)

tap-vsockd: $(DEPS)
	tar cf - $(DEPS) | docker build -t tap-vsockd:build -
	docker run --rm --net=none tap-vsockd:build | tar xf -

clean:
	rm -f tap-vsockd
