# Tag dd4754bf8bc8b4af00210e688f233c869ddaa909
MKSH_IMAGE=mobylinux/mksh@sha256:dd73f387248377af1f900b33eb3422eef093e57f31cdaabad3017fcfb42ef65b
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
