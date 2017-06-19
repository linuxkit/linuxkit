DIRS = $(shell find . -maxdepth 1 -mindepth 1 -type d)
.PHONY: clean dirs $(DIRS)

push: $(DIRS)

$(DIRS):
	$(MAKE) -C $@

clean:
	rm -f hash
