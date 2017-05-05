.PHONY: tag push
IMAGE=test-virtsock

default: push

DEPS=Dockerfile Makefile
SHASUM=alpine:3.5
hash: $(DEPS)
	find $^ -type f | xargs cat  | DOCKER_CONTENT_TRUST=1 docker run --rm -i $(SHASUM) sha1sum | sed 's/ .*//' > $@

tag: hash
	docker pull linuxkit/$(IMAGE):$(shell cat hash) || \
		(docker build --no-cache -t $(IMAGE):build . && \
		 docker tag $(IMAGE):build linuxkit/$(IMAGE):$(shell cat hash))
	docker rmi $(IMAGE):build || true

push: tag
	docker pull linuxkit/$(IMAGE):$(shell cat hash) || \
		docker push linuxkit/$(IMAGE):$(shell cat hash)
	rm -f hash

clean:
	rm -rf hash
	docker rmi $(IMAGE):build || true

.DELETE_ON_ERROR:
