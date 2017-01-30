# Tag: 1ccd40acabb46f43593a960f5d3e0a42df53fb90
GO_COMPILE=mobylinux/go-compile@sha256:208c795a9b6572df09d9bf52ff3cfa97b9a07d109811ec9d28453a22c53f9d17

default: usr/bin/diagnostics-server

DEPS=$(wildcard *.go)

usr/bin/diagnostics-server: $(DEPS) ../vendor/manifest
	mkdir -p $(dir $@)
	tar cf - $(DEPS) -C .. vendor | docker run --rm --net=none --log-driver=none -i $(GO_COMPILE) -o $@ | tar xf -

clean:
	rm -f usr/bin/diagnostics-server

.DELETE_ON_ERROR:
