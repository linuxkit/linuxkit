GO_COMPILE=mobylinux/go-compile:90607983001c2789911afabf420394d51f78ced8@sha256:188beb574d4702a92fa3396a57cabaade28003c82f9413c3121a370ff8becea4

SHA_IMAGE=alpine:3.5@sha256:dfbd4a3a8ebca874ebd2474f044a0b33600d4523d03b0df76e5c5986cb02d7e8

METADATA_BINARY=usr/bin/metadata

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
	rm -rf hash $(DIRS) usr

.DELETE_ON_ERROR:
