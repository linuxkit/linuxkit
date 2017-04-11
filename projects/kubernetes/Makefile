all: build-container-images build-vm-images

build-container-image: Boxfile
	docker run --rm -ti \
	  -v $(PWD):$(PWD) \
	  -v /var/run/docker.sock:/var/run/docker.sock \
	  -w $(PWD) \
	    boxbuilder/box:master Boxfile

push-container-images: build-container-image cache-images
	docker image push mobylinux/kubernetes:latest
	docker image push mobylinux/kubernetes:latest-image-cache-common
	docker image push mobylinux/kubernetes:latest-image-cache-control-plane

build-vm-images: kube-master-initrd.img kube-node-initrd.img

kube-master-initrd.img: kube-master.yml
	../../bin/moby build -name kube-master kube-master.yml

kube-node-initrd.img: kube-node.yml
	../../bin/moby build -name kube-node kube-node.yml

clean:
	rm -f -r \
	  kube-*-bzImage kube-*-cmdline kube-*-disk.img kube-*-initrd.img \
	  image-cache/common image-cache/control-plane

COMMON_IMAGES := \
  kube-proxy-amd64:v1.6.1 \
  k8s-dns-sidecar-amd64:1.14.1 \
  k8s-dns-kube-dns-amd64:1.14.1 \
  k8s-dns-dnsmasq-nanny-amd64:1.14.1 \
  pause-amd64:3.0

CONTROL_PLANE_IMAGES := \
  kube-apiserver-amd64:v1.6.1 \
  kube-controller-manager-amd64:v1.6.1 \
  kube-scheduler-amd64:v1.6.1 \
  etcd-amd64:3.0.17

image-cache/%.tar:
	mkdir -p $(dir $@)
	DOCKER_CONTENT_TRUST=1 docker image pull gcr.io/google_containers/$(shell basename $@ .tar)
	docker image save -o $@ gcr.io/google_containers/$(shell basename $@ .tar)

cache-images:
	for image in $(COMMON_IMAGES) ; \
	  do $(MAKE) "image-cache/common/$${image}.tar" \
	  ; done
	cp image-cache/Dockerfile image-cache/common
	docker image build -t mobylinux/kubernetes:latest-image-cache-common image-cache/common
	for image in $(CONTROL_PLANE_IMAGES) ; \
	  do $(MAKE) "image-cache/control-plane/$${image}.tar" \
	  ; done
	cp image-cache/Dockerfile image-cache/control-plane
	docker image build -t mobylinux/kubernetes:latest-image-cache-control-plane image-cache/control-plane
