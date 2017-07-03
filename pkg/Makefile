DIRS = $(shell find . -maxdepth 1 -mindepth 1 -type d)
.PHONY: clean dirs $(DIRS)

push:
	@set -e; for d in $(DIRS); do make -C "$$d"; done

tag:
	@set -e; for d in $(DIRS); do make -C "$$d" tag; done

clean: ;
