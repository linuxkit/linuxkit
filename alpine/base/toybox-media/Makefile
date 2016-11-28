.PHONY: tag push

BASE=mobylinux/alpine-build-toybox:21d65671879b26ba76d606da625d821455035253
IMAGE=toybox-media

default: push

hash: 
	docker run $(BASE) ls tar sh find sha1sum | \
		docker build -q -t $(IMAGE):build -
	docker run $(IMAGE):build tar cf - bin | docker run -i $(IMAGE):build sha1sum -b - > hash

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
