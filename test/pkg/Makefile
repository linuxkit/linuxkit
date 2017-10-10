DIRS = $(dir $(shell find . -maxdepth 2 -mindepth 2 -type f -name build.yml))
.PHONY: push forcepush build forcebuild show-tag clean

push:
	@set -e; for d in $(DIRS); do linuxkit pkg push "$$d"; done

forcepush:
	@set -e; for d in $(DIRS); do linuxkit pkg push --force "$$d"; done

build:
	@set -e; for d in $(DIRS); do linuxkit pkg build "$$d"; done

forcebuild:
	@set -e; for d in $(DIRS); do linuxkit pkg build --force "$$d"; done

show-tag:
	@set -e; for d in $(DIRS); do linuxkit pkg show-tag "$$d"; done

clean:
