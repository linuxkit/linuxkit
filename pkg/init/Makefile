C_COMPILE=linuxkit/c-compile:63b085bbaec1aa7c42a7bd22a4b1c350d900617d@sha256:286e3a729c7a0b1a605ae150235416190f9f430c29b00e65fa50ff73158998e5
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
	docker pull linuxkit/$(IMAGE):$(shell cat hash) || \
		(docker tag $(IMAGE):build linuxkit/$(IMAGE):$(shell cat hash) && \
		 docker push linuxkit/$(IMAGE):$(shell cat hash))
	docker rmi $(IMAGE):build
	rm -f hash

tag: hash
	docker pull linuxkit/$(IMAGE):$(shell cat hash) || \
		docker tag $(IMAGE):build linuxkit/$(IMAGE):$(shell cat hash)
	docker rmi $(IMAGE):build
	rm -f hash

clean:
	rm -rf hash sbin usr

.DELETE_ON_ERROR:
