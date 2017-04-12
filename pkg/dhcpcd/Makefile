.PHONY: tag push clean container
default: push

IMAGE=dhcpcd

BASE=alpine:3.5

DEPS=dhcpcd.conf usr/lib/dhcpcd/dhcpcd-hooks/10-mtu

container: Dockerfile $(DEPS)
	tar cf - $^ | docker build --no-cache -t $(IMAGE):build -

hash: Dockerfile $(DEPS)
	DOCKER_CONTENT_TRUST=1 docker pull $(BASE)
	tar cf - $^ | docker build --no-cache -t $(IMAGE):build -
	docker run --rm $(IMAGE):build sh -c 'cat $^ /lib/apk/db/installed | sha1sum' | sed 's/ .*//' > hash

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
	rm -rf hash

.DELETE_ON_ERROR:
