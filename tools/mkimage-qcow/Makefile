.PHONY: tag push

IMAGE=mkimage-qcow

default: push

hash: Dockerfile make-qcow
	tar cf - $^ | docker build --no-cache -t $(IMAGE):build -
	docker run --entrypoint sh --rm $(IMAGE):build -c "(cat $^; apt list --installed 2>/dev/null) | sha1sum" | sed 's/ .*//' > hash

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
