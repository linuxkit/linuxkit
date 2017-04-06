CONTAINERD_IMAGE=mobylinux/containerd:a688df6aee1e3700eb8d54dbc81070361df397a2@sha256:59ee3da05fe4dad4fbecff582c86fc30ce75e19a225eeeb07e203c9cc36fe34f
CONTAINERD_BINARIES=usr/bin/containerd usr/bin/containerd-shim usr/bin/ctr usr/bin/dist

RUNC_IMAGE=mobylinux/runc:45884ad2bfad045cbf35f2b78b4c82f75fb19854@sha256:d7c4576405f2318d329f538f847927018d4e8497d7968bd3323ff047e2ffe257
RUNC_BINARY=usr/bin/runc

C_COMPILE=mobylinux/c-compile:81a6bd8ff45d769b60a2ee1acdaccda11ab835c8@sha256:eac250997a3b9784d3285a03c0c8311d4ca6fb63dc75164c987411ba93006487
START_STOP_DAEMON=sbin/start-stop-daemon

default: push

$(RUNC_BINARY): Makefile
	mkdir -p $(dir $@)
	docker run --rm --net=none $(RUNC_IMAGE) tar cf - $@ | tar xf -

$(CONTAINERD_BINARIES): Makefile
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
