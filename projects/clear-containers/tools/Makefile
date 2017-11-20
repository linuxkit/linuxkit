MKFILE_PATH := $(abspath $(lastword $(MAKEFILE_LIST)))
MKFILE_DIR := $(dir $(MKFILE_PATH))
ifdef http_proxy
	BUILD_PROXY = --build-arg http_proxy=$(http_proxy)
	RUN_PROXY = --env http_proxy=$(http_proxy)
endif

ifdef https_proxy
	BUILD_PROXY += --build-arg https_proxy=$(https_proxy)
	RUN_PROXY += --env https_proxy=$(https_proxy)
endif

CMDLINE_FILE=clear-containers-cmdline
KERNEL_FILE=clear-containers-bzImage

IMAGE=qemu-lite

docker_image: Dockerfile
	docker build $(BUILD_PROXY) -t clearcontainers/$(IMAGE) .

qemu-lite: docker_image $(KERNEL_FILE) $(CMDLINE_FILE)
	docker run  --privileged --rm -ti \
		-v $$(pwd)/$(CMDLINE_FILE):/root/clear-containers-cmdline \
		-v $$(pwd)/$(KERNEL_FILE):/root/clear-containers-vmlinux \
		clearcontainers/$(IMAGE) \
		bash

$(KERNEL_FILE):
	$(MKFILE_DIR)/../../../bin/linuxkit build $(MKFILE_DIR)/../clear-containers.yml

push: docker_image
	docker push clearcontainers/$(IMAGE)

clean:
	rm -rf *-bzImage *-cmdline *.img
