RNG_TOOLS_IMAGE=linuxkit/rng-tools:d77676ea6ae229b779868e5eff4600f59b0eb0d8@sha256:61cafe4075e1223a110d8569961bbf9ec0ab64abd4abb04c1677d7ecd7078383
RNGD_BINARY=usr/sbin/rngd

TINI_IMAGE=linuxkit/tini:da0e068f8b537b81f817439a52f63e8f6d91cca3@sha256:bdca5e6703c9553d890c79a4a094e2f05df87f09f6bd5bef26c06d6cba56c112
TINI_BINARY=bin/tini

.PHONY: tag push clean container
default: push

$(TINI_BINARY):
	mkdir -p $(dir $@)
	docker run --rm --net=none $(TINI_IMAGE) tar cf - $@ | tar xf -

$(RNGD_BINARY):
	mkdir -p $(dir $@)
	docker run --rm --net=none $(RNG_TOOLS_IMAGE) tar cf - $@ | tar xf -

SHA_IMAGE=alpine:3.5@sha256:dfbd4a3a8ebca874ebd2474f044a0b33600d4523d03b0df76e5c5986cb02d7e8

IMAGE=rngd

DIRS=dev proc sys
$(DIRS):
	mkdir -p $@

DEPS=$(DIRS) $(TINI_BINARY) $(RNGD_BINARY)

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
	rm -rf hash $(DIRS) usr bin

.DELETE_ON_ERROR:
