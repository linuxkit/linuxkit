DEPS=Dockerfile $(wildcard *.c *.h)

sbin/tap-vsockd: $(DEPS)
	mkdir -p sbin
	tar cf - $(DEPS) | docker build -t tap-vsockd:build -
	docker run --rm --net=none tap-vsockd:build | tar xf - -C sbin

clean:
	rm -rf sbin
