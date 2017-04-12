.PHONY: tag push

BASE=debian:testing
IMAGE=ca-certificates

default: push

hash: Dockerfile
	DOCKER_CONTENT_TRUST=1 docker pull $(BASE)
	tar cf - $^ | docker build --no-cache -t $(IMAGE):build0 -
	docker run --rm $(IMAGE):build0 | docker build --no-cache -t $(IMAGE):build -
	docker run --rm -i $(IMAGE):build0 sh -c "cat /etc/ssl/certs/ca-certificates.crt /etc/debian_version | sha1sum - | sed 's/ .*//'" > $@
	docker rmi $(IMAGE):build0

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
	rm -f hash

.DELETE_ON_ERROR:
