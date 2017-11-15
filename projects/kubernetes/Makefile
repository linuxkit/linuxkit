KUBE_RUNTIME ?= docker
KUBE_NETWORK ?= weave

KUBE_NETWORK_WEAVE ?= v2.0.5

ifeq ($(shell uname -s),Darwin)
KUBE_FORMATS ?= iso-efi
else
KUBE_FORMATS ?= iso-bios
endif

KUBE_FORMAT_ARGS := $(patsubst %,-format %,$(KUBE_FORMATS))

all: build-container-images build-vm-images

build-container-images:
	linuxkit pkg build kubernetes

build-cache-images:
	$(MAKE) -C image-cache build

push-container-images:
	linuxkit pkg push kubernetes
	$(MAKE) -C image-cache push

build-vm-images: kube-master.iso kube-node.iso

kube-master.iso: kube.yml $(KUBE_RUNTIME).yml $(KUBE_RUNTIME)-master.yml $(KUBE_NETWORK).yml
	moby build -name kube-master $(KUBE_FORMAT_ARGS) $^

kube-node.iso: kube.yml $(KUBE_RUNTIME).yml $(KUBE_NETWORK).yml
	moby build -name kube-node $(KUBE_FORMAT_ARGS) $^

weave.yml: kube-weave.yaml

kube-weave.yaml:
	curl -L -o $@ https://cloud.weave.works/k8s/v1.8/net?v=$(KUBE_NETWORK_WEAVE)

clean:
	rm -f -r \
	  kube-*-kernel kube-*-cmdline kube-*-state kube-*-initrd.img *.iso \
	  kube-weave.yaml
	$(MAKE) -C image-cache clean
