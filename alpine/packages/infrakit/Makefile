INFRAKIT_REPO=https://github.com/docker/infrakit.git
INFRAKIT_VERSION=31f74a7
INFRAKIT_COMMIT=31f74a78fbd6b795e87674f712c87dcdabc4035b

BINARIES=infrakit infrakit-flavor-swarm infrakit-flavor-vanilla infrakit-group-default

INFRAKIT_AWS_REPO=https://github.com/docker/infrakit.aws.git
INFRAKIT_AWS_VERSION=ff971ad
INFRAKIT_AWS_COMMIT=ff971ada61e9c4bbe1004f12b5e4a47d79e40c6c

BINARIES_AWS=infrakit-instance-aws

default: usr/bin/infrakit usr/bin/infrakit-instance-aws

usr/bin/infrakit: Dockerfile Makefile
	mkdir -p usr/bin
	BUILD=$$( tar cf - Dockerfile infrakit | \
	  docker build -q --build-arg VERSION=$(INFRAKIT_VERSION) --build-arg REVISION=$(INFRAKIT_COMMIT) - ) && \
	[ -n "$$BUILD" ] && \
	echo "Built $$BUILD" && \
	docker run --net=none --log-driver=none --rm $$BUILD | tar xf - -C usr/bin/ $(BINARIES)

usr/bin/infrakit-instance-aws: Dockerfile.aws Makefile
	mkdir -p usr/bin
	BUILD=$$( tar cf - Dockerfile.aws infrakit.aws | \
	  docker build -q --build-arg VERSION=$(INFRAKIT_AWS_VERSION) --build-arg REVISION=$(INFRAKIT_AWS_COMMIT) -f Dockerfile.aws - ) && \
	[ -n "$$BUILD" ] && \
	echo "Built $$BUILD" && \
	docker run --net=none --log-driver=none --rm $$BUILD | tar xf - -C usr/bin/ $(BINARIES_AWS)

vendor:
	git rm -rf infrakit infrakit.aws || true
	rm -rf infrakit infrakit.aws
	git clone $(INFRAKIT_REPO) && cd infrakit && git checkout $(INFRAKIT_COMMIT)
	git clone $(INFRAKIT_AWS_REPO) && cd infrakit.aws && git checkout $(INFRAKIT_AWS_COMMIT)
	rm -rf infrakit/.git infrakit.aws/.git
	git add infrakit infrakit.aws
	git status

.PHONY: clean
clean:
	rm -rf usr

.DELETE_ON_ERROR:
