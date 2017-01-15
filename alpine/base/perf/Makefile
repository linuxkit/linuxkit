BASE=alpine:3.4

DEPS=Dockerfile

bin/perf: $(DEPS)
	DOCKER_CONTENT_TRUST=1 docker pull $(BASE)
	mkdir -p $(dir $@)
	BUILD=$$( docker build -q . ) && \
	docker run --rm --net=none $$BUILD | tar xf - -C bin

clean:
	rm -rf bin
