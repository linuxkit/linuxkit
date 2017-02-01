TARTAR2INITRD_IMAGE=mobylinux/tartar2initrd:d56cde1558e3080e59a32e3cd7c7141baa601811@sha256:e1ad4522ff906d339da5f250b9ef6bffa5a70b4dec7d2cf7f7dbd0447b79352f

DIRS=$(wildcard */)
.PHONY: clean $(DIRS)

default: container.img

$(DIRS):
	$(MAKE) -C $@

container.img: $(DIRS)
	tar cf - $$(find . -name container.tar) | \
		docker run --rm --read-only --net=none --log-driver=none --tmpfs /tmp -i $(TARTAR2INITRD_IMAGE) > $@

clean:
	for f in $(DIRS); do $(MAKE) -C $$f clean; done
	rm -f container.img
