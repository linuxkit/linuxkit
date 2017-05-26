GO_COMPILE=linuxkit/go-compile:4513068d9a7e919e4ec42e2d7ee879ff5b95b7f5@sha256:bdfadbe3e4ec699ca45b67453662321ec270f2d1a1dbdbf09625776d3ebd68c5

SHA_IMAGE=alpine:3.5@sha256:dfbd4a3a8ebca874ebd2474f044a0b33600d4523d03b0df76e5c5986cb02d7e8

METADATA_BINARY=usr/bin/metadata

ORG?=linuxkit
IMAGE=metadata

.PHONY: tag push clean container
default: push

$(METADATA_BINARY): $(wildcard *.go) Makefile
	mkdir -p $(dir $@)
	tar cf - $^ | docker run --rm --net=none --log-driver=none -i $(GO_COMPILE) -o $@ | tar xf -

DIRS=dev proc sys
$(DIRS):
	mkdir -p $@

DEPS=$(DIRS) $(METADATA_BINARY)

container: Dockerfile $(DEPS)
	tar cf - $^ | docker build --no-cache -t $(IMAGE):build -

hash: Dockerfile $(DEPS)
	find $^ -type f | xargs cat | docker run --rm -i $(SHA_IMAGE) sha1sum - | sed 's/ .*//' > hash

push: hash container
	DOCKER_CONTENT_TRUST=1 docker pull $(ORG)/$(IMAGE):$(shell cat hash) || \
		(docker tag $(IMAGE):build $(ORG)/$(IMAGE):$(shell cat hash) && \
		 DOCKER_CONTENT_TRUST=1 docker push $(ORG)/$(IMAGE):$(shell cat hash))
	docker rmi $(IMAGE):build
	rm -f hash

tag: hash container
	docker pull $(ORG)/$(IMAGE):$(shell cat hash) || \
		docker tag $(IMAGE):build $(ORG)/$(IMAGE):$(shell cat hash)
	docker rmi $(IMAGE):build
	rm -f hash

clean:
	rm -rf hash $(DIRS) usr

.DELETE_ON_ERROR:
