.PHONY: tag push
default: push

ORG?=linuxkit
STAGE_ORG?=linuxkit.datakit.ci:5000
HASH?=$(shell git ls-tree HEAD -- ../$(notdir $(CURDIR)) | awk '{print $$3}')
BASE_DEPS=Dockerfile Makefile
OVERRIDES?=override.yml

UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Linux)
	TIMEOUT=timeout
else
	TIMEOUT=gtimeout
endif

tag: $(BASE_DEPS) $(DEPS)
ifndef $(NETWORK)
	docker build -t $(ORG)/$(IMAGE):$(HASH) .
else
	docker build --network=none -t $(ORG)/$(IMAGE):$(HASH) .
endif

push: tag
	DOCKER_CONTENT_TRUST=1 docker pull $(ORG)/$(IMAGE):$(HASH) || \
	DOCKER_CONTENT_TRUST=1 docker push $(ORG)/$(IMAGE):$(HASH)

hash:
	@echo "$(ORG)/$(IMAGE):$(HASH)"

check:
	@$(TIMEOUT) 5 notary -s https://notary.docker.io lookup docker.io/$(ORG)/$(IMAGE) $(HASH) > /dev/null || \
	$(MAKE) ORG=$(STAGE_ORG) push && \
	echo "- source: $(ORG)/$(IMAGE)\n  subsitute: $(STAGE_ORG)/$(IMAGE):$(HASH)\n" >> $(OVERRIDES)
