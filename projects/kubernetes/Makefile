all: tag-container-images build-vm-images

tag-container-images:
	$(MAKE) -C kubernetes tag

tag-cache-images:
	$(MAKE) -C image-cache tag

push-container-images:
	$(MAKE) -C kubernetes push
	$(MAKE) -C image-cache push

build-vm-images: kube-master-initrd.img kube-node-initrd.img

kube-master-initrd.img: kube-master.yml
	../../bin/moby build -name kube-master kube-master.yml

kube-node-initrd.img: kube-node.yml
	../../bin/moby build -name kube-node kube-node.yml

clean:
	rm -f -r \
	  kube-*-kernel kube-*-cmdline kube-*-state kube-*-initrd.img
	$(MAKE) -C image-cache clean
