.PHONY: tag push
default: push

ORG?=linuxkit
IMAGE=docker-ce
DEPS=Dockerfile

HASH?=$(shell git ls-tree HEAD -- ../$(notdir $(CURDIR)) | awk '{print $$3}')

tag: $(DEPS)
	docker build --squash --no-cache -t $(ORG)/$(IMAGE):$(HASH) .

push: tag
	docker pull $(ORG)/$(IMAGE):$(HASH) || \
	docker push $(ORG)/$(IMAGE):$(HASH)
