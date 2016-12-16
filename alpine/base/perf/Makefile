DEPS=Dockerfile

bin/perf: $(DEPS)
	mkdir -p bin
	BUILD=$$( docker build -q . ) && \
	docker run --rm --net=none $$BUILD | tar xf - -C bin

clean:
	rm -rf bin
