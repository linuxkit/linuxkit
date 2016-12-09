.PHONY: tag push

# Tag: 0b3fe26d29b08f0f74ce68bc6273bbc6cfedef34
BASE=mobylinux/alpine-build-toybox@sha256:5300e0c76a236391131e068c0c40226d4979c9b396f60a1a97cfef4b554476e5
IMAGE=toybox-media

default: push

hash: 
	docker run --rm $(BASE) ls tar sh find sha1sum | \
		docker build -q -t $(IMAGE):build -
	docker run --rm $(IMAGE):build tar cf - bin | docker run -i $(IMAGE):build sha1sum -b - > hash

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
