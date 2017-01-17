# Tag 7a07de557d7f6ae3d72873c32bfb4c51c7687d03
QEMU_IMAGE=mobylinux/qemu-user-static@sha256:cbeba25809c7c3feebc9e20522145e33d8abe5956674afa52814fc57c6644497
QEMU_FILES=qemu-arm-static qemu-aarch64-static qemu-ppc64le-static
QEMU_BINARIES=$(addprefix usr/bin/,$(QEMU_FILES))

# Tag: 6075d4b9c5fe30e19581f1b7ef1813f3041cca32
GO_COMPILE=mobylinux/go-compile@sha256:badfd8a1730ab6e640682d0f95a8f9c51f3cd4b2e8db261fe1a1fd8c6f60bd6e
BINFMT_BINARY=usr/bin/binfmt

# Tag: alpine:3.5
SHA_IMAGE=alpine@sha256:dfbd4a3a8ebca874ebd2474f044a0b33600d4523d03b0df76e5c5986cb02d7e8

IMAGE=binfmt

.PHONY: tag push clean container
default: push

$(QEMU_BINARIES):
	mkdir -p $(dir $@)
	docker run --rm --net=none $(QEMU_IMAGE) tar cf - $@ | tar xf -

$(BINFMT_BINARY): main.go
	mkdir -p $(dir $@)
	tar cf - $^ | docker run --rm --net=none --log-driver=none -i $(GO_COMPILE) -o $@ | tar xf -
	
DIRS=dev proc sys
$(DIRS):
	mkdir -p $@

DEPS=$(DIRS) $(QEMU_BINARIES) $(BINFMT_BINARY) etc/binfmt.d/00_moby.conf

container: Dockerfile $(DEPS)
	tar cf - $^ | docker build --no-cache -t $(IMAGE):build -

hash: Dockerfile $(DEPS)
	tar cf - $^ | docker run --rm -i $(SHA_IMAGE) sha1sum - | sed 's/ .*//' > hash

push: hash container
	docker pull mobylinux/$(IMAGE):$(shell cat hash) || \
		(docker tag $(IMAGE):build mobylinux/$(IMAGE):$(shell cat hash) && \
		 docker push mobylinux/$(IMAGE):$(shell cat hash))
	docker rmi $(IMAGE):build
	rm -f hash

tag: hash container
	docker pull mobylinux/$(IMAGE):$(shell cat hash) || \
		docker tag $(IMAGE):build mobylinux/$(IMAGE):$(shell cat hash)
	docker rmi $(IMAGE):build
	rm -f hash

clean:
	rm -rf hash $(DIRS) usr

.DELETE_ON_ERROR:
