KERNEL_IMAGE=iad.ocir.io/odx-fn/linuxkit/kernel:4.14.48-ee964ca1fb8641370812be8ef4592b2708d3551b-amd64

default: ebpf.tag

ebpf.tag: Dockerfile
	BUILD=$$( docker build -q --build-arg IMAGE=$(KERNEL_IMAGE) . ) && [ -n "$$BUILD" ] && echo "Built $$BUILD" && \
	echo $$BUILD > $@

clean:
	rm -f ebpf.tag

.DELETE_ON_ERROR:
