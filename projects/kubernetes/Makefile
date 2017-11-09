KUBE_RUNTIME ?= docker
KUBE_NETWORK ?= weave-v2.0.5

INIT_YAML ?=
INIT_YAML += network.yaml

ifeq ($(shell uname -s),"Darwin")
KUBE_FORMATS ?= iso-efi
endif
KUBE_FORMATS ?= iso-bios

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

# NB cannot use $^ because $(INIT_YAML) is not for consumption by "moby build"
kube-master.iso: kube.yml $(KUBE_RUNTIME).yml $(KUBE_RUNTIME)-master.yml $(INIT_YAML)
	moby build -name kube-master $(KUBE_FORMAT_ARGS) kube.yml $(KUBE_RUNTIME).yml $(KUBE_RUNTIME)-master.yml

kube-node.iso: kube.yml $(KUBE_RUNTIME).yml
	moby build -name kube-node $(KUBE_FORMAT_ARGS) $^

network.yaml: $(KUBE_NETWORK).yaml
	ln -nf $< $@

weave-%.yaml:
	curl -L -o $@ https://cloud.weave.works/k8s/v1.8/net?v=$*

clean:
	rm -f -r \
	  kube-*-kernel kube-*-cmdline kube-*-state kube-*-initrd.img *.iso \
	  weave-*.yaml network.yaml
	$(MAKE) -C image-cache clean
