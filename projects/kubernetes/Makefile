all: tag-container-images build-vm-images

tag-container-images:
	$(MAKE) -C kubernetes tag

tag-cache-images:
	$(MAKE) -C image-cache tag

push-container-images:
	$(MAKE) -C kubernetes push
	$(MAKE) -C image-cache push

build-vm-images: kube-master.iso kube-node.iso

kube-master.iso: kube-master.yml
	moby build -name kube-master -output iso-efi -output iso-bios kube-master.yml

kube-node.iso: kube-node.yml
	moby build -name kube-node -output iso-efi -output iso-bios kube-node.yml

clean:
	rm -f -r \
	  kube-*-kernel kube-*-cmdline kube-*-state kube-*-initrd.img *.iso
	$(MAKE) -C image-cache clean
