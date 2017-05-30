.PHONY: tag push
default: push

ORG?=linuxkit
IMAGE=qemu
DEPS=Dockerfile Makefile

HASH?=$(shell git ls-tree HEAD -- ../$(notdir $(CURDIR)) | awk '{print $$3}')

tag: $(DEPS)
	DOCKER_CONTENT_TRUST=1 docker build --no-cache --network=none -t $(ORG)/$(IMAGE):$(HASH) .

push: tag
	docker pull $(ORG)/$(IMAGE):$(HASH) || \
	docker push $(ORG)/$(IMAGE):$(HASH)
