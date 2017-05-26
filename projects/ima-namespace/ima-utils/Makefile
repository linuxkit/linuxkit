.PHONY: tag push
default: push

IMAGE=ima-utils
DEPS=Dockerfile
ORG?=linuxkit

HASH?=$(shell git ls-tree HEAD -- ../$(notdir $(CURDIR)) | awk '{print $$3}')

tag: $(DEPS)
	docker pull $(ORG)/$(IMAGE):$(HASH) || \
	docker build --no-cache -t $(ORG)/$(IMAGE):$(HASH) .

push: tag
	docker pull $(ORG)/$(IMAGE):$(HASH) || \
	docker push $(ORG)/$(IMAGE):$(HASH)
