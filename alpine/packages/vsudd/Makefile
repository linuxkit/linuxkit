# Tag: 470f68ec32e016484fb8e0bcc2f06cd3f201ea68
GO_COMPILE=mobylinux/go-compile@sha256:b723c0e95a6293300392e57d6ab52574f9217e2410b390a24cbfc4f9070edb6b

default: sbin/vsudd

DEPS=$(wildcard *.go)

sbin/vsudd: $(DEPS) ../vendor/manifest
	mkdir -p $(dir $@)
	tar cf - $(DEPS) -C .. vendor | docker run --rm --net=none --log-driver=none -i $(GO_COMPILE) -o $@ | tar xf -

clean:
	rm -rf sbin

.DELETE_ON_ERROR:
