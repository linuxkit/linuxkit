DEPS=Dockerfile $(wildcard *.c *.h)

sbin/add-random: $(DEPS)
	mkdir -p sbin
	tar cf - $(DEPS) | docker build -t random:build -
	docker run --rm --net=none random:build | tar xf - -C sbin

clean:
	rm -rf sbin
