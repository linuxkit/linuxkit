CONTAINERD_IMAGE=mobylinux/containerd:6fac214f15f3dffc1c77d82874d828483b5ff0d0@sha256:59f9b78506a00c77c0fbde857a9dceb7812b9f4387c5d09ddc751ef7a42ae632
CONTAINERD_BINARIES=usr/bin/containerd usr/bin/containerd-shim usr/bin/ctr usr/bin/dist

RUNC_IMAGE=mobylinux/runc:e6426f67d17d7e34dceded8b1ab45454b41652e2@sha256:fe2014f51c18e2b0ef48c3ddfe7fac0d35547c59de5f3bc32b10c70ed123cdc6
RUNC_BINARY=usr/bin/runc

C_COMPILE=mobylinux/c-compile:81a6bd8ff45d769b60a2ee1acdaccda11ab835c8@sha256:eac250997a3b9784d3285a03c0c8311d4ca6fb63dc75164c987411ba93006487
START_STOP_DAEMON=sbin/start-stop-daemon

default: push

$(RUNC_BINARY):
	mkdir -p $(dir $@)
	docker run --rm --net=none $(RUNC_IMAGE) tar cf - $@ | tar xf -

$(CONTAINERD_BINARIES):
	mkdir -p $(dir $@)
	docker run --rm --net=none $(CONTAINERD_IMAGE) tar cf - $@ | tar xf -

$(START_STOP_DAEMON): start-stop-daemon.c
	mkdir -p $(dir $@)
	tar cf - $^ | docker run --rm --net=none --log-driver=none -i $(C_COMPILE) -o $@ | tar xf -

.PHONY: tag push

BASE=alpine:3.5
IMAGE=init

ETC=$(shell find etc -type f)

hash: Dockerfile $(ETC) init $(RUNC_BINARY) $(CONTAINERD_BINARIES) $(START_STOP_DAEMON)
	DOCKER_CONTENT_TRUST=1 docker pull $(BASE)
	tar cf - $^ | docker build --no-cache -t $(IMAGE):build -
	docker run --rm $(IMAGE):build sh -c 'cat $^ /lib/apk/db/installed | sha1sum' | sed 's/ .*//' > $@

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

clean:
	rm -rf hash sbin usr

.DELETE_ON_ERROR:
