QEMU_IMAGE=linuxkit/qemu-user-static:da39a3ee5e6b4b0d3255bfef95601890afd80709@sha256:65ee2b44b35c9457d83884c292b46bc3b5558a13af2c8eb187322f09160131a4
QEMU_FILES=qemu-arm qemu-aarch64 qemu-ppc64le
QEMU_BINARIES=$(addprefix usr/bin/,$(QEMU_FILES))

GO_COMPILE=linuxkit/go-compile:90607983001c2789911afabf420394d51f78ced8@sha256:8b6566c6fd9f3bca31191b919449248d3cb1ca3a562276fca7199e93451d6596

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
	docker pull linuxkit/$(IMAGE):$(shell cat hash) || \
		(docker tag $(IMAGE):build linuxkit/$(IMAGE):$(shell cat hash) && \
		 docker push linuxkit/$(IMAGE):$(shell cat hash))
	docker rmi $(IMAGE):build
	rm -f hash

tag: hash container
	docker pull linuxkit/$(IMAGE):$(shell cat hash) || \
		docker tag $(IMAGE):build linuxkit/$(IMAGE):$(shell cat hash)
	docker rmi $(IMAGE):build
	rm -f hash

clean:
	rm -rf hash $(DIRS) usr

.DELETE_ON_ERROR:
