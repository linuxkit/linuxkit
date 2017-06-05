.PHONY: tag push
default: push

ORG?=linuxkit
IMAGE=getty
DEPS=Dockerfile usr/bin/rungetty.sh

HASH?=$(shell git ls-tree HEAD -- ../$(notdir $(CURDIR)) | awk '{print $$3}')

hash:
	@echo $(HASH)

tag: $(DEPS)
	docker build --squash --no-cache --network=none -t $(ORG)/$(IMAGE):$(HASH) .

test-tag:
	docker tag $(ORG)/$(IMAGE):$(HASH) getty:test

push: tag
	DOCKER_CONTENT_TRUST=1 docker pull $(ORG)/$(IMAGE):$(HASH) || \
	DOCKER_CONTENT_TRUST=1 docker push $(ORG)/$(IMAGE):$(HASH)
