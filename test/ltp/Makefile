LTP_VERSION=20170116

all: ltp.tar push

# Build LTP and get the result as a tarball
DEPS=Dockerfile.build Makefile
ltp.tag: $(DEPS)
	BUILD=$$(docker build -f $< -q . --build-arg LTP_VERSION=$(LTP_VERSION)) && [ -n "$$BUILD" ] && echo "Built $$BUILD" && echo "$$BUILD" > $@

ltp.tar: ltp.tag
	docker run --rm --net=none --log-driver=none $(shell cat ltp.tag) tar cf - opt/ltp  > $@

SHASUM=alpine:3.5
IMAGE=test-ltp-$(LTP_VERSION)

# Note: We do not compute the hash from all the dependencies here
# because the ltp binaries will change everytime we build. Ideally, we
# would calculate the hash from the source and the apt-get cache, but
# it's not that critical.
hash: Dockerfile.pkg ltp.tar check.sh $(DEPS)
	tar xf ltp.tar
	tar cf - Dockerfile.pkg opt check.sh | docker build --no-cache -t $(IMAGE):build -f Dockerfile.pkg -
	cat Dockerfile.pkg check.sh $(DEPS) | DOCKER_CONTENT_TRUST=1 docker run --rm -i $(SHASUM) sha1sum | sed 's/ .*//' > $@

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

.PHONY: clean
clean:
	rm -rf opt ltp.tar ltp.tag hash

.DELETE_ON_ERROR:
