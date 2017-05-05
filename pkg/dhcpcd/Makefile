.PHONY: tag push
default: push

IMAGE=dhcpcd
DEPS=Dockerfile Makefile dhcpcd.conf usr/lib/dhcpcd/dhcpcd-hooks/10-mtu

HASH?=$(shell git ls-tree HEAD -- ../$(notdir $(CURDIR)) | awk '{print $$3}')

tag: $(DEPS)
	docker build --squash --no-cache --network=none -t linuxkit/$(IMAGE):$(HASH) .

push: tag
	docker pull linuxkit/$(IMAGE):$(HASH) || \
	docker push linuxkit/$(IMAGE):$(HASH)
