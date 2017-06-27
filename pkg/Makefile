DIRS = $(shell find . -type d -depth 1)
export OVERRIDES = $(CURDIR)/override.yml

.PHONY: clean dirs check hash $(DIRS)

push:
	@set -e; for d in $(DIRS); do make -C "$$d"; done

tag:
	@set -e; for d in $(DIRS); do make -C "$$d" tag; done

check:
	@set -e; for d in $(DIRS); do make -C "$$d" check; done

hash:
	@set -e; for d in $(DIRS); do make -C "$$d" hash; done

clean:
	rm -f $(OVERRIDES)
