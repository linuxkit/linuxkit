CONTAINERD_IMAGE=mobylinux/containerd:c9c8a069da6dccd2803ab476ee0d57a8768f0dcb@sha256:ff5aa0b1086e8c600d6e1508cfae4da31e4935d36ec40f0128aa73113b664e7f
CONTAINERD_BINARIES=usr/bin/containerd usr/bin/containerd-shim usr/bin/ctr usr/bin/dist

RUNC_IMAGE=mobylinux/runc:f1cee12a65e7b7de06a01aec24609dc3175e1542@sha256:ff1ead6aa4388418ee07f8e93304e5b2fa9b975fe2399474d408654a1411a44a
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
