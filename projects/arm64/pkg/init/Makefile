C_COMPILE=mor1/c-compile:d8aa4bec68dcd4e6a48762ed9de09ee8051ed2d3
START_STOP_DAEMON=sbin/start-stop-daemon

default: push

$(START_STOP_DAEMON): start-stop-daemon.c
	mkdir -p $(dir $@)
	tar cf - $^ | docker run --rm --net=none --log-driver=none -i $(C_COMPILE) -o $@ | tar xf -

.PHONY: tag push

ORG=mor1
BASE=aarch64/alpine:3.5
IMAGE=init

ETC=$(shell find etc -type f)

hash: Dockerfile $(ETC) init $(START_STOP_DAEMON)
	DOCKER_CONTENT_TRUST=0 docker pull $(BASE)
	tar cf - $^ | docker build --no-cache -t $(IMAGE):build -
	docker run --rm $(IMAGE):build sh -c 'cat $^ /lib/apk/db/installed | sha1sum' | sed 's/ .*//' > $@

push: hash
	docker pull $(ORG)/$(IMAGE):$(shell cat hash) || \
		(docker tag $(IMAGE):build $(ORG)/$(IMAGE):$(shell cat hash) && \
		 docker push $(ORG)/$(IMAGE):$(shell cat hash))
	docker rmi $(IMAGE):build
	rm -f hash

tag: hash
	docker pull $(ORG)/$(IMAGE):$(shell cat hash) || \
		docker tag $(IMAGE):build $(ORG)/$(IMAGE):$(shell cat hash)
	docker rmi $(IMAGE):build
	rm -f hash

clean:
	rm -rf hash sbin usr

.DELETE_ON_ERROR:
