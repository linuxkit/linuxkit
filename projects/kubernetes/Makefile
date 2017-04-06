all: build-container-images build-vm-images

build-container-images: Boxfile
	docker run --rm -ti \
	  -v $(PWD):$(PWD) \
	  -v /var/run/docker.sock:/var/run/docker.sock \
	  -w $(PWD) \
	    boxbuilder/box:master Boxfile

push-container-images: build-container-image
	docker push errordeveloper/mobykube:master

build-vm-images:
	../../bin/moby build -name kube-master kube-master.yml
