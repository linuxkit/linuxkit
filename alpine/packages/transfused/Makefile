DEPS=Dockerfile $(wildcard *.c *.h)

transfused: $(DEPS)
	tar cf - $(DEPS) | docker build -t transfused:build -
	docker run --rm --net=none transfused:build | tar xf -

clean:
	rm -f transfused
