# Tag 48a34c44de78a7607b8b03e54efffc8ede461342
MKSH_IMAGE=mobylinux/mksh@sha256:b9ca6f8ec3d7855db91162555128429030e0ebc950881c1f09015c40d9e875d2
MKSH_FILE=mksh
MKSH_PATH=/bin

# Tag: 41e4b91c9a619e46f76ce2d024067c09b62f07b4
CACERT_IMAGE=mobylinux/ca-certificates@sha256:6ca2dca9cfb8534a55f3a17f8797943527db5bbac08c98a5c9a4836250f4c548
CACERT_FILE=ca-certificates.crt
CACERT_PATH=/etc/ssl/certs

default: rootfs

$(MKSH_FILE):
	docker run --rm --net=none $(MKSH_IMAGE) tar cf - -C $(MKSH_PATH) $(MKSH_FILE) | tar xf -

$(CACERT_FILE):
	docker run --rm --net=none $(CACERT_IMAGE) tar cf - -C $(CACERT_PATH) $(CACERT_FILE) | tar xf -

rootfs: Dockerfile test.sh $(MKSH_FILE) $(CACERT_FILE)
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
	rm -rf rootfs $(MKSH_FILE) $(CACERT_FILE)

.DELETE_ON_ERROR:
