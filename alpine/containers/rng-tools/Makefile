# Tag c530ee985d21d6c0ee495d03086fc408b8183466
TINI_IMAGE=mobylinux/tini@sha256:065d5a16baf022324ac5fb997908f3efa5cc21982b455e832aed1151eb636a2a
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
