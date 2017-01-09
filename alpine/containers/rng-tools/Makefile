# Tag 7cb780fd6b60c089964e81efd6553853c491d59f
TINI_IMAGE=mobylinux/tini@sha256:7da8c5b371e0d7d3fb1778e96c0bc634e39ace7bf1e7a73bffbf1f8360127fdb
TINI_BINARY=tini

default: rootfs

$(TINI_BINARY): Dockerfile
	docker run --rm --net=none $(TINI_IMAGE) tar cf - -C /bin $@ | tar xf -

EXCLUDE=--exclude .dockerenv --exclude Dockerfile \
	--exclude dev/console --exclude dev/pts --exclude dev/shm \
	--exclude etc/hostname --exclude etc/hosts --exclude etc/mtab --exclude etc/resolv.conf

rootfs: Dockerfile fix-textrels-on-PIC-x86.patch sha256sums $(TINI_BINARY)
	mkdir -p $@
	BUILD=$$( tar cf - $^ | docker build -q - ) && \
	[ -n "$$BUILD" ] && \
	echo "Built $$BUILD" && \
	IMAGE=$$( docker run --rm --net=none $$BUILD | docker build -q - ) && \
	[ -n "$$IMAGE" ] && \
	echo "Built $$IMAGE" && \
	CONTAINER=$$( docker create $$IMAGE /dev/null ) && \
	docker export $$CONTAINER | tar -xf - -C $@ $(EXCLUDE) && \
	docker rm $$CONTAINER

clean:
	rm -rf rootfs $(TINI_BINARY)

.DELETE_ON_ERROR:
