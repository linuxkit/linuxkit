DIRS = $(dir $(shell find . -maxdepth 2 -mindepth 2 -type f -name build.yml))
# These have more complex builds which have not yet been ported to linuxkit pkg.
# No more entries should be added to this list.
# Note that these are only recursed on for the push target which was the historical behaviour.
MAKEDIRS = alpine guestfs

.PHONY: push forcepush build forcebuild show-tag clean

push:
	@set -e; for d in $(DIRS); do linuxkit pkg push "$$d"; done
	@set -e; for d in $(MAKEDIRS); do make -C "$$d" push; done

forcepush:
	@set -e; for d in $(DIRS); do linuxkit pkg push --force "$$d"; done

build:
	@set -e; for d in $(DIRS); do linuxkit pkg build "$$d"; done

forcebuild:
	@set -e; for d in $(DIRS); do linuxkit pkg build --force "$$d"; done

show-tag:
	@set -e; for d in $(DIRS); do linuxkit pkg show-tag "$$d"; done

clean:
