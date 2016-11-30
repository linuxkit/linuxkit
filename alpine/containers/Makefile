DIRS=$(wildcard */)
.PHONY: clean $(DIRS)

default: $(DIRS)

$(DIRS):
	$(MAKE) -C $@

clean:
	for f in $(DIRS); do $(MAKE) -C $$f clean; done
