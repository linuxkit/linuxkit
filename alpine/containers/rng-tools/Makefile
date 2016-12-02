# Tag 7b94dce736818ca5c9d5367be360b79714687ca5
TINI_IMAGE=mobylinux/tini@sha256:5f697e501ce12af1c72fbdf5dd74299bcc8c4f58e6215a7c48627dc6e11d9a29
TINI_BINARY=tini

default: rootfs

$(TINI_BINARY): Dockerfile
	docker run --rm --net=none $(TINI_IMAGE) tar cf - -C /bin $@ | tar xf -

rootfs: Dockerfile fix-textrels-on-PIC-x86.patch sha256sums $(TINI_BINARY)
	mkdir -p $@
	BUILD=$$( tar cf - $^ | docker build -q - ) && \
	[ -n "$$BUILD" ] && \
	echo "Built $$BUILD" && \
	IMAGE=$$( docker run --rm --net=none $$BUILD | docker build -q - ) && \
	[ -n "$$IMAGE" ] && \
	echo "Built $$IMAGE" && \
	CONTAINER=$$( docker create $$IMAGE /dev/null ) && \
	docker export $$CONTAINER | tar -xf - -C $@ && \
	docker rm $$CONTAINER && \
	( cd $@ && rm -rf .dockerenv Dockerfile dev/* etc/hostname etc/hosts etc/mtab etc/resolv.conf )

clean:
	rm -rf rootfs $(TINI_BINARY)

.DELETE_ON_ERROR:
