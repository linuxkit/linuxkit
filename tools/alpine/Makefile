.PHONY: build push

ORG?=linuxkit
IMAGE=alpine
DEPS=packages

ARCH := $(shell uname -m)
ifeq ($(ARCH), x86_64)
DEPS += packages.x86_64
endif
ifeq ($(ARCH), aarch64)
DEPS += packages.aarch64
endif
ifeq ($(ARCH), s390x)
DEPS += packages.s390x
endif

ORG?=linuxkit
IMAGE?=alpine
TAG?=$(shell git ls-tree --full-tree HEAD -- $(PWD) | awk '{print $$3}')
DIRTY?=$(shell git diff-index HEAD -- $(PWD))
ifneq ($(DIRTY),)
TAG := $(TAG)-dirty
endif


default: push

show-tag:
	@echo "$(ORG)/$(IMAGE):$(TAG)"

iid: Dockerfile Makefile $(DEPS)
	docker build --no-cache --iidfile iid .

hash: Makefile iid
	docker run --rm $(shell cat iid) cat /etc/alpine-hash-arch > $@

versions.$(ARCH): Makefile hash iid
	echo "# $(ORG)/$(IMAGE):$(shell cat hash)" > versions.$(ARCH)
	docker run --rm $(shell cat iid) find /mirror -name '*.apk' -exec basename '{}' .apk \; | LANG=C sort | (echo '# automatically generated list of installed packages'; cat -) >> versions.$(ARCH)

push: hash iid versions.$(ARCH)
	docker pull $(ORG)/$(IMAGE):$(shell cat hash) || \
		(docker tag $(shell cat iid) $(ORG)/$(IMAGE):$(shell cat hash) && \
		 docker push $(ORG)/$(IMAGE):$(shell cat hash))
	rm -f iid

push-manifest:
	./push-manifest.sh $(ORG) $(IMAGE)

build: hash iid versions.$(ARCH)
	docker pull $(ORG)/$(IMAGE):$(shell cat hash) || \
		docker tag $(shell cat iid) $(ORG)/$(IMAGE):$(shell cat hash)
	rm -f iid

clean:
	rm -f hash iid
