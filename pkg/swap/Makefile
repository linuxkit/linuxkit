# copy from mount
.PHONY: tag push clean

BASE=alpine:3.5
IMAGE=linuxkit/swap

default: push

hash: Dockerfile swap.sh
	DOCKER_CONTENT_TRUST=1 docker pull $(BASE)
	tar cf - $^ | docker build --no-cache -t $(IMAGE):build -
	docker run --rm --entrypoint /bin/sh $(IMAGE):build -c "cat $^ /lib/apk/db/installed | sha1sum" | sed 's/ .*//' > $@

push: hash
	docker pull $(IMAGE):$(shell cat hash) || \
		(docker tag $(IMAGE):build $(IMAGE):$(shell cat hash) && \
		 docker push $(IMAGE):$(shell cat hash))
	docker rmi $(IMAGE):build
	rm -f hash

tag: hash
	docker pull $(IMAGE):$(shell cat hash) || \
		docker tag $(IMAGE):build $(IMAGE):$(shell cat hash)
	docker rmi $(IMAGE):build
	rm -f hash

clean:
	rm -f hash

.DELETE_ON_ERROR:
