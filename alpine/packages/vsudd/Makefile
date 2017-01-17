# Tag: bcc7cf6e6fc75e22dacc467043d462e5e9a3dc3f
GO_COMPILE=mobylinux/go-compile@sha256:f0f952162aff1d8962921da101ac18e21d7bb0cb53a3b36a4209a040da5f00e9

default: sbin/vsudd

DEPS=$(wildcard *.go)

sbin/vsudd: $(DEPS) ../vendor/manifest
	mkdir -p $(dir $@)
	tar cf - $(DEPS) -C .. vendor | docker run --rm --net=none --log-driver=none -i $(GO_COMPILE) -o $@ | tar xf -

clean:
	rm -rf sbin

.DELETE_ON_ERROR:
