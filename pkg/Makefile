DIRS = $(shell find . -type d -depth 1)
.PHONY: clean dirs $(DIRS)

push:
	@set -e; for d in $(DIRS); do make -C "$$d"; done

tag:
	@set -e; for d in $(DIRS); do make -C "$$d" tag; done

clean: ;
