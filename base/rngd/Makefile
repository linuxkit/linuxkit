RNG_TOOLS_IMAGE=mobylinux/rng-tools:b6aed437bad8f1f4471b11f1affe3420eaf5d42f@sha256:8e74e6a39b072ebee65ee4b83ebf224787afb473ea250c897dd24fa43b387d06
RNGD_BINARY=usr/sbin/rngd

TINI_IMAGE=mobylinux/tini:6b25b62f4d893de8721fd2581411039b17e8a253@sha256:39b4a459018ffc155a9fcbbf952fa625c77f5a8d7599b326eade529d3dc723fc
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
