ORG?=linuxkitprojects
IMAGE=kubernetes-e2e
HASH?=$(shell git ls-tree HEAD -- ../$(notdir $(CURDIR)) | awk '{print $$3}')

kubernetes_version=v1.8.0

kube_release_artefacts=https://dl.k8s.io/$(kubernetes_version)/bin/linux/amd64

.PHONY: build
build: _cache/kubernetes-test.tar.gz _cache/kubectl
	docker build --network=none -t $(ORG)/$(IMAGE):$(HASH) .

.PHONY: local
local: build
	docker run -ti -v /var/lib/kubeadm:/etc/kubernetes $(ORG)/$(IMAGE):$(HASH)

_cache/kubectl:
	mkdir -p _cache
	curl -fSL -o $@ https://dl.k8s.io/$(kubernetes_version)/bin/linux/amd64/kubectl

_cache/kubernetes-test.tar.gz:
	mkdir -p _cache
	curl -fSL -o $@ https://dl.k8s.io/$(kubernetes_version)/kubernetes-test.tar.gz

default: build

clean:
	rm -rf _cache

