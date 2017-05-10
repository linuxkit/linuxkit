.PHONY: tag push

ORG?=linuxkit
IMAGE=qemu-ga

HASH?=$(shell git ls-tree HEAD -- ../$(notdir $(CURDIR)) | awk '{print $$3}')

default: push

tag: Dockerfile
	docker build --network=none -t $(ORG)/$(IMAGE):$(HASH) .

push: tag
	docker pull $(ORG)/$(IMAGE):$(HASH) || docker push $(ORG)/$(IMAGE):$(HASH)
