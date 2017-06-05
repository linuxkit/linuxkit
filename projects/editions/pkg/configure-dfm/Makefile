default=tag
ORG?=edtestorg
IMAGE=configure-dfm
DEPS=Dockerfile run.sh

HASH?=$(shell git ls-tree HEAD -- ../$(notdir $(CURDIR)) | awk '{print $$3}')

tag: $(DEPS)
	tar -cf - $(DEPS) | docker build --no-cache --network=none -t $(ORG)/$(IMAGE):$(HASH) -

push: tag
	docker pull $(ORG)/$(IMAGE):$(HASH) || \
	docker push $(ORG)/$(IMAGE):$(HASH)
