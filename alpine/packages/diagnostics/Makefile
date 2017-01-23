# Tag: 56f72369b2100961d418cd8a614a6019bb53dc9c
GO_COMPILE=mobylinux/go-compile@sha256:98eb2ded03c6bb717b5b19c5723f221b00e85c18352c042d0c8470a4fb32ea24

default: usr/bin/diagnostics-server

DEPS=$(wildcard *.go)

usr/bin/diagnostics-server: $(DEPS) ../vendor/manifest
	mkdir -p $(dir $@)
	tar cf - $(DEPS) -C .. vendor | docker run --rm --net=none --log-driver=none -i $(GO_COMPILE) -o $@ | tar xf -

clean:
	rm -f usr/bin/diagnostics-server

.DELETE_ON_ERROR:
