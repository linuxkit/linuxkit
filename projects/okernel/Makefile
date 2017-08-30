DEBUG ?= 0

all: image tag

# For IMAGE_VERSION, the "<n>" must be increased whenever
# the kernel config or the patches change. We don't expect this to
# happen very often as the minor version number gets update quite
# frequently.
#
# IMAGE_VERSION is used to determine if a new image should be pushed to hub.

ifdef KERNEL
ifneq ($(KERNEL),latest)
KERNEL_VERSION=$(KERNEL)
IMAGE_VERSION=$(KERNEL_VERSION)-0
else
KERNEL_VERSION=linux-okernel
IMAGE_VERSION=latest
endif
else
KERNEL_VERSION=linux-okernel
IMAGE_VERSION=latest
endif

IMAGE=okernel
DEPS=Dockerfile.okernel Makefile kernel_config.okernel Dockerfile.kvmod

ifdef HTTP_PROXY
	PROXY_ARG = --build-arg HTTP_PROXY=$(HTTP_PROXY)
endif

ifdef HTTPS_PROXY
	PROXY_ARG += --build-arg HTTPS_PROXY=$(HTTPS_PROXY)
endif

.PHONY: image push tag kvmod
image: $(DEPS)
	BUILD=$$( tar cf - $^ | docker build -f $< $(PROXY_ARG) --build-arg DEBUG=$(DEBUG) --build-arg KERNEL_VERSION=$(KERNEL_VERSION) -q -t $(IMAGE):build - ) && [ -n "$$BUILD" ] && echo "Built $$BUILD" && echo "$$BUILD" > build_info


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
	docker rmi $(IMAGE):build
	rm -f hash

.PHONY: clean
clean:
	rm -f build_info

.DELETE_ON_ERROR:
