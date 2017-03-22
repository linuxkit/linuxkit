.PHONY: tag push clean container
all: push

# Build the virtsock stress server
VIRTSOCK_BINARY=bin/virtsock_stress
DEPS=Dockerfile.build Makefile
virtsock.tag: $(DEPS)
	BUILD=$$(docker build -f $< -q .) && [ -n "$$BUILD" ] && echo "Built $$BUILD" && echo "$$BUILD" > $@

$(VIRTSOCK_BINARY): virtsock.tag
	mkdir -p $(dir $@)
	docker run --rm --net=none --log-driver=none $(shell cat virtsock.tag) > $@
	chmod ugo+x $@

TINI_IMAGE=mobylinux/tini:6b25b62f4d893de8721fd2581411039b17e8a253@sha256:39b4a459018ffc155a9fcbbf952fa625c77f5a8d7599b326eade529d3dc723fc
TINI_BINARY=bin/tini

$(TINI_BINARY):
	mkdir -p $(dir $@)
	docker run --rm --net=none $(TINI_IMAGE) tar cf - $@ | tar xf -

SHASUM=alpine:3.5
IMAGE=test-virtsock
container: Dockerfile.pkg $(DEPS) $(VIRTSOCK_BINARY) $(TINI_BINARY)
	tar cf - $^ | docker build --no-cache -t $(IMAGE):build -f $< -

hash: Dockerfile.pkg $(DEPS)
	find $^ -type f | xargs cat  | DOCKER_CONTENT_TRUST=1 docker run --rm -i $(SHASUM) sha1sum | sed 's/ .*//' > $@

push: hash container
	docker pull mobylinux/$(IMAGE):$(shell cat hash) || \
		(docker tag $(IMAGE):build mobylinux/$(IMAGE):$(shell cat hash) && \
		docker push mobylinux/$(IMAGE):$(shell cat hash))
	docker rmi $(IMAGE):build
	rm -f hash

tag: hash container
	docker pull mobylinux/$(IMAGE):$(shell cat hash) || \
		docker tag $(IMAGE):build mobylinux/$(IMAGE):$(shell cat hash)
	docker rmi $(IMAGE):build
	rm -f hash

clean:
	rm -rf $(VIRTSOCK_BINARY) $(TINI_BINARY) virtsock.tag hash

.DELETE_ON_ERROR:
