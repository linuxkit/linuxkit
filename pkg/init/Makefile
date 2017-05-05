.PHONY: tag push
default: push

IMAGE=init
DEPS=Dockerfile init $(wildcard etc/*) $(wildcard etc/init.d/*)

HASH?=$(shell git ls-tree HEAD -- ../$(notdir $(CURDIR)) | awk '{print $$3}')

tag: $(DEPS)
	docker build --no-cache --network=none -t linuxkit/$(IMAGE):$(HASH) .

push: tag
	docker pull linuxkit/$(IMAGE):$(HASH) || \
	docker push linuxkit/$(IMAGE):$(HASH)
