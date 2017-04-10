GO_COMPILE=mobylinux/go-compile:3afebc59c5cde31024493c3f91e6102d584a30b9@sha256:e0786141ea7df8ba5735b63f2a24b4ade9eae5a02b0e04c4fca33b425ec69b0a

SHA_IMAGE=alpine:3.5@sha256:dfbd4a3a8ebca874ebd2474f044a0b33600d4523d03b0df76e5c5986cb02d7e8

SYSFS_BINARY=usr/bin/sysfs

IMAGE=sysfs

.PHONY: tag push clean container
default: push

$(SYSFS_BINARY): main.go
	mkdir -p $(dir $@)
	tar cf - $^ | docker run --rm --net=none --log-driver=none -i $(GO_COMPILE) -o $@ | tar xf -

DIRS=dev proc sys
$(DIRS):
	mkdir -p $@

DEPS=$(DIRS) $(SYSFS_BINARY) etc/sysfs.d/00-moby.conf

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
