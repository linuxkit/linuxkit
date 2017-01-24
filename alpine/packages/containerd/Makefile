# Tag: cd9f1788bb2e8e7a342f32dd8ba70f450ff7ae2c
CONTAINERD_IMAGE=mobylinux/containerd@sha256:96a96f60751a1bb762e285f36e0cf9c5b4126933bb91475b7369efc990539165
CONTAINERD_BINARIES=usr/bin/containerd usr/bin/containerd-shim usr/bin/ctr

# Tag: 1a72d872ae97b90ae52482e4cb33a2778e168358
RUNC_IMAGE=mobylinux/runc@sha256:2ea83d68baa57d547ec6d407d09cd495e4ef9cd6b0207ec3525fb2752b4a197e
RUNC_BINARY=usr/bin/runc

default: $(RUNC_BINARY) $(CONTAINERD_BINARIES)

$(RUNC_BINARY):
	mkdir -p $(dir $@)
	docker run --rm --net=none $(RUNC_IMAGE) tar cf - $@ | tar xf -

$(CONTAINERD_BINARIES):
	mkdir -p $(dir $@)
	docker run --rm --net=none $(CONTAINERD_IMAGE) tar cf - $@ | tar xf -

clean:
	rm -rf usr

.DELETE_ON_ERROR:
