DEPS=Dockerfile $(wildcard *.c *.h)

sbin/transfused: $(DEPS)
	mkdir -p sbin
	tar cf - $(DEPS) | docker build -t transfused:build -
	docker run --rm --net=none transfused:build | tar xf - -C sbin

clean:
	rm -rf sbin
