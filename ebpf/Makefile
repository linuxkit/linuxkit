KERNEL_FILES=-C ../alpine/kernel/x86_64 kernel-headers.tar kernel-dev.tar kernel-modules.tar

default: ebpf.tag

ebpf.tag: Dockerfile
	BUILD=$$( tar cf - $^ $(KERNEL_FILES) | docker build -q - ) && [ -n "$$BUILD" ] && echo "Built $$BUILD" && \
	echo $$BUILD > $@

clean:
	rm -f ebpf.tag

.DELETE_ON_ERROR:
