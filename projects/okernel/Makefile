all: image tag

IMAGE=okernel
OKERNEL_REPO=https://github.com/linux-okernel/linux-okernel
DEPS=Dockerfile.okernel Makefile kernel_config.okernel Dockerfile.kvmod

DEBUG ?= 0

ifdef KERNEL
ifneq ($(KERNEL),latest)
ifneq (,$(findstring ok-,$(KERNEL)))
KERNEL_VERSION=$(KERNEL)
IMAGE_VERSION=$(KERNEL_VERSION)
LATEST=0
else
$(error Error, please specify a valid okernel kernel version beginning with "ok-")
endif
else
KERNEL_VERSION=linux-okernel
IMAGE_VERSION=$(shell git ls-remote --tags $(OKERNEL_REPO) ok-\* | awk -F"/" '{print $$3}' | tail -1)
LATEST=1
endif
else
KERNEL_VERSION=linux-okernel
IMAGE_VERSION=$(shell git ls-remote --tags $(OKERNEL_REPO) ok-\* | awk -F"/" '{print $$3}' | tail -1)
LATEST=1
endif



ifdef HTTP_PROXY
	PROXY_ARG = --build-arg HTTP_PROXY=$(HTTP_PROXY)
endif

ifdef HTTPS_PROXY
	PROXY_ARG += --build-arg HTTPS_PROXY=$(HTTPS_PROXY)
endif

.PHONY: image push tag kvmod
image: $(DEPS)
	BUILD=$$( tar cf - $^ | docker build --no-cache -f $< $(PROXY_ARG) --build-arg DEBUG=$(DEBUG) --build-arg KERNEL_VERSION=$(KERNEL_VERSION) -q -t $(IMAGE):build - ) && [ -n "$$BUILD" ] && echo "Built $$BUILD" && echo "$$BUILD" > build_info


default: tag

kvmod: 
	docker build --no-cache --build-arg IMAGE=$(IMAGE):$(IMAGE_VERSION) -t okernel-kvmod:$(IMAGE_VERSION) - < Dockerfile.kvmod

push: image
	(docker tag $(IMAGE):build $(IMAGE):$(IMAGE_VERSION) && \
	 docker push $(IMAGE):$(IMAGE_VERSION)) 
	docker rmi $(IMAGE):build
	rm -f hash

tag: image
	docker tag $(IMAGE):build $(IMAGE):$(IMAGE_VERSION)
ifeq (1,$(LATEST))
	docker tag $(IMAGE):build $(IMAGE):latest
endif
	docker rmi $(IMAGE):build
	rm -f hash

.PHONY: clean
clean:
	rm -f build_info

.DELETE_ON_ERROR:
