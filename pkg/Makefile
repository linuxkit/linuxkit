DIRS = $(dir $(shell find . -maxdepth 2 -mindepth 2 -type f -name build.yml))
.PHONY: push force-push build forcebuild show-tag clean

OPTIONS ?=

push:
	@set -e; for d in $(DIRS); do linuxkit pkg push $(OPTIONS) "$$d"; done

forcepush:
	@set -e; for d in $(DIRS); do linuxkit pkg push $(OPTIONS) --force "$$d"; done

build:
	@set -e; for d in $(DIRS); do linuxkit pkg build $(OPTIONS) "$$d"; done

forcebuild:
	@set -e; for d in $(DIRS); do linuxkit pkg build $(OPTIONS) --force "$$d"; done

show-tag:
	@set -e; for d in $(DIRS); do linuxkit pkg show-tag $(OPTIONS) "$$d"; done

clean:
