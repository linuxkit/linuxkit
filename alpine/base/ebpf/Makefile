KERNEL_FILES=-C ../../kernel/x86_64 kernel-headers.tar kernel-dev.tar kernel-modules.tar

default: ebpf.tag

ebpf.tag: Dockerfile 100-musl-compat.patch bcc-gnuism.patch decl.patch intl.patch temp_failure.patch cdefs.h error.h
	BUILD=$$( tar cf - $^ $(KERNEL_FILES) | docker build -q - ) && [ -n "$$BUILD" ] && echo "Built $$BUILD" && \
	echo $$BUILD > $@

clean:
	rm -f ebpf.tag

.DELETE_ON_ERROR:
