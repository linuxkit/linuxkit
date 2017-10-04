DIRS = $(shell find . -maxdepth 1 -mindepth 1 -type d)
.PHONY: clean dirs $(DIRS)

# TODO(ijc) once all switch to `linuxkit pkg` update this to:
# linuxkit pkg <build|push> [--force] $$d
# linuxkit pkg show-tag $$d

push:
	@set -e; for d in $(DIRS); do make -C "$$d"; done

tag:
	@set -e; for d in $(DIRS); do make -C "$$d" tag; done

forcetag:
	@set -e; for d in $(DIRS); do make -C "$$d" forcetag; done

show-tag:
	@set -e; for d in $(DIRS); do make -C "$$d" show-tag; done

clean: ;
