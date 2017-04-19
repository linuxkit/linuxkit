.PHONY: tag push
ORG=mor1
BASE=mor1/alpine-build-toybox:f830f77db6a50393157e80200eee8a23865a7b12
IMAGE=toybox-media

default: push

hash:
	docker run --rm $(BASE) ls tar sh find sha1sum | \
		docker build -q -t $(IMAGE):build -
	docker run --rm $(IMAGE):build tar cf - bin | docker run -i $(IMAGE):build sha1sum -b - > hash

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
	rm -f hash

.DELETE_ON_ERROR:
