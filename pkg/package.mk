.PHONY: tag push
default: push

ORG?=linuxkit
HASH?=$(shell git ls-tree HEAD -- ../$(notdir $(CURDIR)) | awk '{print $$3}')
BASE_DEPS=Dockerfile Makefile

tag: $(BASE_DEPS) $(DEPS)
ifndef $(NETWORK)
	docker build -t $(ORG)/$(IMAGE):$(HASH) .
else
	docker build --network=none -t $(ORG)/$(IMAGE):$(HASH) .
endif

push: tag
	DOCKER_CONTENT_TRUST=1 docker pull $(ORG)/$(IMAGE):$(HASH) || \
	DOCKER_CONTENT_TRUST=1 docker push $(ORG)/$(IMAGE):$(HASH)
