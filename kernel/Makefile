DEBUG ?= 0

all:	bzImage push

ifdef AUFS
DEPS=Dockerfile.aufs kernel_config kernel_config.debug kernel_config.aufs patches-4.9
else
ifdef LTS4.4
DEPS=Dockerfile.4.4 kernel_config kernel_config.debug kernel_config.4.4 patches-4.4
else
DEPS=Dockerfile kernel_config kernel_config.debug patches-4.9
endif
endif
kernel.tag: $(DEPS)
	BUILD=$$( tar cf - $^ | docker build -f $< --build-arg DEBUG=$(DEBUG) -q - ) && [ -n "$$BUILD" ] && echo "Built $$BUILD" && echo "$$BUILD" > $@

bzImage: kernel.tag
	rm -rf etc/kernel-patches
	mkdir -p x86_64 etc lib usr sbin etc/kernel-patches
	docker run --rm --net=none --log-driver=none $(shell cat kernel.tag) tar cf - bzImage kernel-dev.tar kernel-headers.tar vmlinux kernel-modules.tar | tar xf - -C x86_64
	cp x86_64/kernel-modules.tar kernel.tar
	cp x86_64/bzImage $@

.PHONY: push tag

MEDIA_TOYBOX=mobylinux/toybox-media:0a26fe5f574e444849983f9c4148ef74b3804d55@sha256:5ac38f77b66deb194c9016591b9b096e81fcdc9f7c3e6d01566294a6b4b4ebd2
SHASUM=alpine:3.5

BASE="$MEDIA_TOYBOX"
IMAGE=kernel

default: push

Dockerfile.media:
	printf "FROM $(MEDIA_TOYBOX)\nADD . /\n" > $@

hash: Dockerfile.media bzImage kernel.tar $(DEPS)
	tar cf - $^ | docker build --no-cache -t $(IMAGE):build -f Dockerfile.media -
	tar cf - $^ | DOCKER_CONTENT_TRUST=1 docker run --rm -i $(SHASUM) sha1sum | sed 's/ .*//' > $@

push: hash
	docker pull mobylinux/$(IMAGE):$(shell cat hash) || \
		(docker tag $(IMAGE):build mobylinux/$(IMAGE):$(shell cat hash) && \
		 docker push mobylinux/$(IMAGE):$(shell cat hash))
	docker rmi $(IMAGE):build
	rm -f hash

tag: hash
	docker pull mobylinux/$(IMAGE):$(shell cat hash) || \
		docker tag $(IMAGE):build mobylinux/$(IMAGE):$(shell cat hash)
	docker rmi $(IMAGE):build
	rm -f hash

.PHONY: clean
clean:
	rm -rf x86_64 lib usr sbin kernel.tag hash Dockerfile.media bzImage kernel.tar

.DELETE_ON_ERROR:
