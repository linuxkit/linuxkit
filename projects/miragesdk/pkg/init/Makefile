C_COMPILE=mobylinux/c-compile:81a6bd8ff45d769b60a2ee1acdaccda11ab835c8@sha256:eac250997a3b9784d3285a03c0c8311d4ca6fb63dc75164c987411ba93006487
START_STOP_DAEMON=sbin/start-stop-daemon

default: push

$(START_STOP_DAEMON): start-stop-daemon.c
	mkdir -p $(dir $@)
	tar cf - $^ | docker run --rm --net=none --log-driver=none -i $(C_COMPILE) -o $@ | tar xf -

.PHONY: tag push

BASE=alpine:3.5
IMAGE=init

ETC=$(shell find etc -type f)

hash: Dockerfile $(ETC) init $(START_STOP_DAEMON)
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
