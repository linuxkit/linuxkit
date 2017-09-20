KUBE_RUNTIME ?= docker

WEAVE_VERSION := v2.0.4

INIT_YAML ?=
INIT_YAML += weave.yaml

all: tag-container-images build-vm-images

tag-container-images:
	$(MAKE) -C kubernetes tag

tag-cache-images:
	$(MAKE) -C image-cache tag

push-container-images:
	$(MAKE) -C kubernetes push
	$(MAKE) -C image-cache push

build-vm-images: kube-master.iso kube-node.iso

# NB cannot use $^ because $(INIT_YAML) is not for consumption by "moby build"
kube-master.iso: kube.yml $(KUBE_RUNTIME).yml $(KUBE_RUNTIME)-master.yml $(INIT_YAML)
	moby build -name kube-master -format iso-efi -format iso-bios kube.yml $(KUBE_RUNTIME).yml $(KUBE_RUNTIME)-master.yml

kube-node.iso: kube.yml $(KUBE_RUNTIME).yml
	moby build -name kube-node -format iso-efi -format iso-bios $^

weave.yaml:
	curl -L -o $@ https://cloud.weave.works/k8s/v1.7/net?v=$(WEAVE_VERSION)

clean:
	rm -f -r \
	  kube-*-kernel kube-*-cmdline kube-*-state kube-*-initrd.img *.iso
	$(MAKE) -C image-cache clean
