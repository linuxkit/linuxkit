SHA_IMAGE=aarch64/alpine:3.5

.PHONY: tag push

ORG=mor1
BASE=mor1/golang-arm64:c76abc12b23b502648b0603066f42cbeacd2f310
IMAGE=runc

default: push

hash: Dockerfile
	DOCKER_CONTENT_TRUST=0 docker pull $(BASE)
	tar cf - $^ | docker build --no-cache -t $(IMAGE):build0 -
	docker run --rm $(IMAGE):build0 | docker build --no-cache -t $(IMAGE):build -
	docker rmi $(IMAGE):build0
	find $^ -type f | xargs cat | docker run --rm -i $(SHA_IMAGE) sha1sum - | sed 's/ .*//' > $@

push: hash
	docker pull mor1/$(IMAGE):$(shell cat hash) || \
		(docker tag $(IMAGE):build mor1/$(IMAGE):$(shell cat hash) && \
		 docker push mor1/$(IMAGE):$(shell cat hash))
	docker rmi $(IMAGE):build
	rm -f hash

tag: hash
	docker pull mor1/$(IMAGE):$(shell cat hash) || \
		docker tag $(IMAGE):build mor1/$(IMAGE):$(shell cat hash)
	docker rmi $(IMAGE):build
	rm -f hash

clean:
	rm -f hash

.DELETE_ON_ERROR:
