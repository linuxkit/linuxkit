QEMU_IMAGE=mobylinux/qemu-user-static:da39a3ee5e6b4b0d3255bfef95601890afd80709@sha256:6c022f700dc6c263c9107c08aa29b4eae3b43b7b7594b7be3e421f69b51f53e5
QEMU_FILES=qemu-arm qemu-aarch64 qemu-ppc64le
QEMU_BINARIES=$(addprefix usr/bin/,$(QEMU_FILES))

GO_COMPILE=mobylinux/go-compile:d2d25ac665b5148ad356d0eab3ff3762a68c633d@sha256:aab55d0c317460850e66a07dd94139cc11ea9e1c0bee88716a6a8c768740885f

BINFMT_BINARY=usr/bin/binfmt

SHA_IMAGE=alpine:3.5@sha256:dfbd4a3a8ebca874ebd2474f044a0b33600d4523d03b0df76e5c5986cb02d7e8

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
	find $^ -type f | xargs cat | docker run --rm -i $(SHA_IMAGE) sha1sum - | sed 's/ .*//' > hash

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
