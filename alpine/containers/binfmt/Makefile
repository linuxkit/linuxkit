# Tag 7a07de557d7f6ae3d72873c32bfb4c51c7687d03
QEMU_IMAGE=mobylinux/qemu-user-static@sha256:cbeba25809c7c3feebc9e20522145e33d8abe5956674afa52814fc57c6644497
QEMU_BINARIES=qemu-arm-static qemu-aarch64-static qemu-ppc64le-static

default: rootfs

$(QEMU_BINARIES):
	docker run --rm --net=none $(QEMU_IMAGE) tar cf - -C /usr/bin $@ | tar xf -

rootfs: Dockerfile main.go 00_moby.conf $(QEMU_BINARIES)
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
	rm -rf rootfs $(QEMU_BINARIES)

.DELETE_ON_ERROR:
