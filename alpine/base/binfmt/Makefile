.PHONY: tag push

IMAGE=binfmt

# Tag 7a07de557d7f6ae3d72873c32bfb4c51c7687d03
QEMU_IMAGE=mobylinux/qemu-user-static@sha256:cbeba25809c7c3feebc9e20522145e33d8abe5956674afa52814fc57c6644497
QEMU_BINARIES=qemu-arm-static qemu-aarch64-static qemu-ppc64le-static

default: push

hash: Dockerfile 00_moby.conf main.go $(QEMU_BINARIES)
	tar cf - $^ | docker build --no-cache -t $(IMAGE):build -
	docker run --rm --entrypoint sh $(IMAGE):build -c 'cat usr/bin/binfmt etc/binfmt.d/00_moby.conf | sha1sum' | sed 's/ .*//' > hash

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

$(QEMU_BINARIES):
	docker run --rm --net=none $(QEMU_IMAGE) tar cf - -C /usr/bin $@ | tar xf -

clean:
	rm -f hash

.DELETE_ON_ERROR:
