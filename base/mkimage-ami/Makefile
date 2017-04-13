.PHONY: tag push

IMAGE=mkimage-ami

default: push

hash: Dockerfile make-ami
	tar cf - $^ | docker build --no-cache -t $(IMAGE):build -
	docker run --rm $(IMAGE):build sh -c "(cat $^; apt list --installed 2>/dev/null) | sha1sum" | sed 's/ .*//' > hash

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
	rm -f hash

.DELETE_ON_ERROR:
