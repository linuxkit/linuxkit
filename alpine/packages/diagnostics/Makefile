# Tag: 9c777a22fd84f08e1ea342c0b4ebabfb09fca086
GO_COMPILE=mobylinux/go-compile@sha256:5e7f1909b0316261653d84f73aeb9188ef20d06b1ecf92eaa4402b32221adfba

default: usr/bin/diagnostics-server

DEPS=$(wildcard *.go)

usr/bin/diagnostics-server: $(DEPS) ../vendor/manifest
	mkdir -p $(dir $@)
	tar cf - $(DEPS) -C .. vendor | docker run --rm --net=none --log-driver=none -i $(GO_COMPILE) -o $@ | tar xf -

clean:
	rm -f usr/bin/diagnostics-server

.DELETE_ON_ERROR:
