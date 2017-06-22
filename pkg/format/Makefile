.PHONY: tag push
default: push

ORG?=linuxkit
IMAGE=format
DEPS=Dockerfile format.sh

HASH?=$(shell git ls-tree HEAD -- ../$(notdir $(CURDIR)) | awk '{print $$3}')

shellcheck:
	docker run --rm -it -v $(PWD):/mnt nlknguyen/alpine-shellcheck *.sh

tag: $(DEPS) shellcheck
	docker build --squash --no-cache --network=none -t $(ORG)/$(IMAGE):$(HASH) .

push: tag
	DOCKER_CONTENT_TRUST=1 docker pull $(ORG)/$(IMAGE):$(HASH) || \
	DOCKER_CONTENT_TRUST=1 docker push $(ORG)/$(IMAGE):$(HASH)
