KUBE_RUNTIME ?= docker

all: tag-container-images build-vm-images

tag-container-images:
	$(MAKE) -C kubernetes tag

tag-cache-images:
	$(MAKE) -C image-cache tag

push-container-images:
	$(MAKE) -C kubernetes push
	$(MAKE) -C image-cache push

build-vm-images: kube-master.iso kube-node.iso

kube-master.iso: kube-master.yml $(KUBE_RUNTIME).yml $(KUBE_RUNTIME)-master.yml
	moby build -name kube-master -format iso-efi -format iso-bios kube-master.yml $(KUBE_RUNTIME).yml $(KUBE_RUNTIME)-master.yml

kube-node.iso: kube-node.yml $(KUBE_RUNTIME).yml
	moby build -name kube-node -format iso-efi -format iso-bios kube-node.yml $(KUBE_RUNTIME).yml

clean:
	rm -f -r \
	  kube-*-kernel kube-*-cmdline kube-*-state kube-*-initrd.img *.iso
	$(MAKE) -C image-cache clean
